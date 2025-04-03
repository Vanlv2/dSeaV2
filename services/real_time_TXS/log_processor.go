package real_time_TXS

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func process_handle_log(vLog types.Log, chainName string) {
	processLock.Lock()
	defer processLock.Unlock()

	chainData := GetChainData(chainName)
	if chainData == nil {
		return
	}

	blockNumber := big.NewInt(int64(vLog.BlockNumber))
	txHash := vLog.TxHash.Hex()

	txKey := fmt.Sprintf("%d-%s-%d", vLog.BlockNumber, txHash, vLog.Index)

	// Kiểm tra reorg
	if blockNumber.Cmp(chainData.LastProcessedBlock) <= 0 {
		if !chainData.IsProcessingReorg {
			fmt.Printf("⚠️ Phát hiện reorg trên chain %s, đang xử lý lại từ khối %d\n",
				chainName, blockNumber)

			select {
			case chainData.DisconnectedChannel <- struct{}{}:
				chainData.IsProcessingReorg = true
			default:
				fmt.Printf("Channel đã đầy, bỏ qua tín hiệu reorg\n")
			}
		}
		return
	}

	// Kiểm tra xem giao dịch đã được xử lý chưa
	if chainData.ProcessedTxs[txKey] {
		return
	}

	// Chuyển đổi log thành map và lưu trữ
	txData := extract_transaction_data(&vLog, chainName)

	// Thêm giao dịch vào mảng toàn cục
	if txData != nil {
		AddTransaction(txData)
	}

	// Đánh dấu giao dịch đã được xử lý
	chainData.ProcessedTxs[txKey] = true

	// Làm sạch map nếu quá lớn
	if len(chainData.ProcessedTxs) > 100000 {
		fmt.Printf("⚠️ Map processedTransactions cho %s quá lớn (%d), đang làm sạch...\n",
			chainName, len(chainData.ProcessedTxs))
		cutoffBlock := blockNumber.Uint64() - 1000
		clean_processed_transactions(chainName, cutoffBlock)
	}

	// Cập nhật khối cuối cùng được xử lý
	if !chainData.IsProcessingReorg && blockNumber.Cmp(chainData.LastProcessedBlock) > 0 {
		chainData.LastProcessedBlock.Set(blockNumber)
	}
}

func handle_logs(ctx context.Context, client *ethclient.Client, chainName string) {
	logs := make(chan types.Log, 100)
	var retryDelay time.Duration = 1 * time.Second
	maxRetryDelay := 30 * time.Second
	maxRetries := 10
	retryCount := 0

	sessionProcessed := make(map[string]bool)

	chainData := GetChainData(chainName)
	if chainData == nil {
		return
	}

	// Khởi tạo LastProcessedBlock bằng khối mới nhất nếu chưa được khởi tạo
	latestBlock, err := client.BlockNumber(ctx)
	if err == nil {
		processLock.Lock()
		// Chỉ cập nhật nếu LastProcessedBlock đang là 0
		if chainData.LastProcessedBlock.Cmp(big.NewInt(0)) == 0 {
			chainData.LastProcessedBlock = big.NewInt(int64(latestBlock))
		}
		processLock.Unlock()
	}

	for {
		if ctx.Err() != nil {
			fmt.Println("Context đã kết thúc, dừng xử lý logs")
			return
		}

		sessionProcessed = make(map[string]bool)
		subCtx, cancel := context.WithCancel(ctx)
		processorDone := make(chan struct{})

		go func() {
			defer close(processorDone)
			for {
				select {
				case vLog, ok := <-logs:
					if !ok {
						return
					}

					logKey := fmt.Sprintf("%d-%s-%d", vLog.BlockNumber, vLog.TxHash.Hex(), vLog.Index)

					if sessionProcessed[logKey] {
						continue
					}

					sessionProcessed[logKey] = true
					process_handle_log(vLog, chainName)

				case <-subCtx.Done():
					return
				}
			}
		}()

		err := subscribe_to_logs(subCtx, client, logs, chainName)
		cancel()

		<-processorDone

		if err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Println("Context bị hủy, dừng lắng nghe logs")
				return
			}

			retryCount++
			jitter := time.Duration(100+int64(time.Now().UnixNano()%900)) * time.Millisecond
			currentRetryDelay := retryDelay + jitter

			if retryCount > maxRetries {
				fmt.Printf("Đã vượt quá số lần thử lại tối đa (%d), đang reset bộ đếm\n", maxRetries)
				retryCount = 0
			}

			fmt.Printf("Lỗi kết nối log cho %s: %v. Thử lại sau %v... (lần thử: %d/%d)\n",
				chainName, err, currentRetryDelay, retryCount, maxRetries)

			timer := time.NewTimer(currentRetryDelay)
			select {
			case <-timer.C:
				retryDelay = min(retryDelay*2, maxRetryDelay)
			case <-ctx.Done():
				timer.Stop()
				fmt.Println("Context bị hủy trong khi đợi thử lại")
				return
			}
		} else {
			retryDelay = 1 * time.Second
			retryCount = 0
		}
	}
}

func subscribe_to_logs(ctx context.Context, client *ethclient.Client, logs chan types.Log, chainName string) error {
	chainData := GetChainData(chainName)
	if chainData == nil {
		return fmt.Errorf("không tìm thấy dữ liệu cho chain %s", chainName)
	}

	latestBlock, err := client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("không thể lấy số khối hiện tại: %v", err)
	}

	// Cập nhật LastProcessedBlock nếu chưa được khởi tạo
	processLock.Lock()
	if chainData.LastProcessedBlock.Cmp(big.NewInt(0)) == 0 {
		chainData.LastProcessedBlock = big.NewInt(int64(latestBlock))
	}
	processLock.Unlock()

	currentBlock := big.NewInt(int64(latestBlock))
	query := create_query(chainName, currentBlock, nil)

	// Thử sử dụng WebSocket nếu có
	var sub ethereum.Subscription
	var subErr error

	if chainData.Config.WssRPC != "" {
		// Thử kết nối WebSocket
		wsClient, wsErr := ethclient.Dial(chainData.Config.WssRPC)
		if wsErr == nil {
			sub, subErr = wsClient.SubscribeFilterLogs(ctx, query, logs)
			if subErr == nil {
				fmt.Printf("✅ Đã thiết lập subscription thành công qua WebSocket, đang lắng nghe sự kiện...\n")
				defer sub.Unsubscribe()

				for {
					select {
					case err := <-sub.Err():
						return fmt.Errorf("lỗi subscription: %v", err)
					case <-ctx.Done():
						fmt.Println("Context đã kết thúc, dừng subscription")
						return ctx.Err()
					}
				}
			}
			fmt.Printf("⚠️ Không thể đăng ký nhận logs qua WebSocket: %v, thử phương pháp polling...\n", subErr)
		}
	}

	// Đặt ticker để poll logs định kỳ
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastCheckedBlock := currentBlock

	for {
		select {
		case <-ticker.C:
			newLatestBlock, err := client.BlockNumber(ctx)
			if err != nil {
				fmt.Printf("⚠️ Không thể lấy số khối mới nhất: %v\n", err)
				continue
			}

			if big.NewInt(int64(newLatestBlock)).Cmp(lastCheckedBlock) > 0 {
				fromBlock := new(big.Int).Add(lastCheckedBlock, big.NewInt(1))
				toBlock := big.NewInt(int64(newLatestBlock))

				pollQuery := create_query(chainName, fromBlock, toBlock)
				pollCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				newLogs, err := client.FilterLogs(pollCtx, pollQuery)
				cancel()

				if err != nil {
					fmt.Printf("⚠️ Lỗi khi poll logs: %v\n", err)
				} else {
					for _, log := range newLogs {
						select {
						case logs <- log:
							// Log đã được gửi thành công
						case <-ctx.Done():
							return ctx.Err()
						}
					}
				}

				lastCheckedBlock = toBlock
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func create_query(chainName string, fromBlock, toBlock *big.Int) ethereum.FilterQuery {
	chainData := GetChainData(chainName)
	if chainData == nil {
		return ethereum.FilterQuery{}
	}

	// Đảm bảo địa chỉ hợp đồng được chuẩn hóa thành chữ thường
	chainData.Config.WrappedBTCAddress = strings.ToLower(chainData.Config.WrappedBTCAddress)

	var topics [][]common.Hash

	// if chainData.Config.TransferSignature != "" {
	// 	topics = [][]common.Hash{
	// 		{common.HexToHash(chainData.Config.TransferSignature)},
	// 	}
	// }

	addresses := []common.Address{
		common.HexToAddress(chainData.Config.WrappedBTCAddress),
		common.HexToAddress(chainData.Config.WrapWrappedBNBAddress),
	}

	return ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Addresses: addresses,
		Topics:    topics,
	}
}

func extract_transaction_data(logData *types.Log, chainName string) map[string]interface{} {
	txData := make(map[string]interface{})

	// Phần code hiện tại của bạn
	txData["name_chain"] = chainName
	txData["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	txData["block_number"] = logData.BlockNumber
	txData["tx_hash"] = logData.TxHash.Hex()
	txData["address"] = logData.Address.Hex()
	txData["log_index"] = logData.Index

	chainData := GetChainData(chainName)
	if chainData != nil {
		tokenAddresses := map[string]string{
			strings.ToLower(chainData.Config.WrappedBTCAddress):     "WBTC",
			strings.ToLower(chainData.Config.WrapWrappedBNBAddress): "WBNB",
		}

		// Kiểm tra nếu địa chỉ log trùng với bất kỳ địa chỉ token nào
		if symbol, exists := tokenAddresses[strings.ToLower(logData.Address.Hex())]; exists {
			txData["tokenSymbol"] = symbol
		}
	}

	amount := new(big.Int).SetBytes(logData.Data)
	txData["amount"] = amount.String()
	txData["raw_data"] = fmt.Sprintf("%x", logData.Data)

	if len(logData.Topics) > 0 {
		eventSignature := logData.Topics[0].Hex()
		txData["event_signature"] = eventSignature

		if transactionType, err := Parse_event_signature_name(eventSignature); err == nil {
			txData["transaction_type"] = transactionType
		} else {
			fmt.Printf("Không thể parse event signature %s: %v\n", eventSignature, err)
			txData["transaction_type"] = "Unknown"
		}

		for i, topic := range logData.Topics {
			txData[fmt.Sprintf("topic_%d", i)] = topic.Hex()
		}
	}

	if len(logData.Topics) > 1 {
		fromAddr := common.HexToAddress(logData.Topics[1].Hex()).Hex()
		txData["from_address"] = fromAddr
	}

	if len(logData.Topics) > 2 {
		toAddr := common.HexToAddress(logData.Topics[2].Hex()).Hex()
		txData["to_address"] = toAddr
	}

	// Thêm mới: Tính giá trị USD nếu có tokenSymbol
	if _, hasSymbol := txData["tokenSymbol"]; hasSymbol {
		err := EnrichTransactionWithUSDValue(txData)
		if err != nil {
			fmt.Printf("Không thể tính giá trị USD: %v\n", err)
		}
	}

	// Lưu dữ liệu vào ChainData
	if chainData != nil {
		chainData.LogData = txData
	} else {
		fmt.Printf("Cảnh báo: Không tìm thấy dữ liệu cho chain %s\n", chainName)
	}
	return txData
}
