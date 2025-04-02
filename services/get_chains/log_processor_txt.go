package main

import (
	"context"
	"errors"
	"fmt"
	"log"
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
		log.Printf("Không tìm thấy dữ liệu cho chain %s", chainName)
		return
	}

	blockNumber := big.NewInt(int64(vLog.BlockNumber))
	txHash := vLog.TxHash.Hex()

	txKey := fmt.Sprintf("%d-%s-%d", vLog.BlockNumber, txHash, vLog.Index)

	logMap := log_to_map(&vLog, chainName)
	chainData.LogData = logMap

	log.Printf("💼 GIAO DỊCH: Khối=%d, TxHash=%s, Địa chỉ=%s, Index=%d",
		vLog.BlockNumber, txHash, vLog.Address.Hex(), vLog.Index)

	if blockNumber.Cmp(chainData.LastProcessedBlock) <= 0 {
		if !chainData.IsProcessingReorg {
			log.Printf("⚠️ Phát hiện reorg trên chain %s, đang xử lý lại từ khối %d",
				chainName, blockNumber)

			select {
			case chainData.DisconnectedChannel <- struct{}{}:
				chainData.IsProcessingReorg = true
			default:
				log.Printf("Channel đã đầy, bỏ qua tín hiệu reorg")
			}
		}
		return
	}

	if chainData.ProcessedTxs[txKey] {
		return
	}

	if len(chainData.ProcessedTxs) > 100000 {
		log.Printf("⚠️ Map processedTransactions cho %s quá lớn (%d), đang làm sạch...",
			chainName, len(chainData.ProcessedTxs))
		cutoffBlock := blockNumber.Uint64() - 1000
		clean_processed_transactions(chainName, cutoffBlock)
	} else {
		chainData.ProcessedTxs[txKey] = true
	}

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
		log.Fatalf("Không tìm thấy dữ liệu cho chain %s", chainName)
	}

	for {
		if ctx.Err() != nil {
			log.Println("Context đã kết thúc, dừng xử lý logs")
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

		log.Printf("Thiết lập subscription mới cho logs từ khối mới nhất (chain: %s)", chainName)
		err := subscribe_to_logs(subCtx, client, logs, chainName)
		cancel()

		<-processorDone

		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("Context bị hủy, dừng lắng nghe logs")
				return
			}

			retryCount++
			jitter := time.Duration(100+int64(time.Now().UnixNano()%900)) * time.Millisecond
			currentRetryDelay := retryDelay + jitter

			if retryCount > maxRetries {
				log.Printf("Đã vượt quá số lần thử lại tối đa (%d), đang reset bộ đếm", maxRetries)
				retryCount = 0
			}

			log.Printf("Lỗi kết nối log cho %s: %v. Thử lại sau %v... (lần thử: %d/%d)",
				chainName, err, currentRetryDelay, retryCount, maxRetries)

			timer := time.NewTimer(currentRetryDelay)
			select {
			case <-timer.C:
				retryDelay = min(retryDelay*2, maxRetryDelay)
			case <-ctx.Done():
				timer.Stop()
				log.Println("Context bị hủy trong khi đợi thử lại")
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

	log.Printf("🔄 Bắt đầu theo dõi từ block mới nhất: %d cho chain %s", latestBlock, chainName)
	currentBlock := big.NewInt(int64(latestBlock))
	query := create_query(chainName, currentBlock, nil)

	sub, err := client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return fmt.Errorf("không thể đăng ký nhận logs: %v", err)
	}
	defer sub.Unsubscribe()

	log.Printf("✅ Đã thiết lập subscription thành công, đang lắng nghe sự kiện...")

	for {
		select {
		case err := <-sub.Err():
			return fmt.Errorf("lỗi subscription: %v", err)
		case <-ctx.Done():
			log.Println("Context đã kết thúc, dừng subscription")
			return ctx.Err()
		}
	}
}

func create_query(chainName string, fromBlock, toBlock *big.Int) ethereum.FilterQuery {
	chainData := GetChainData(chainName)
	if chainData == nil {
		log.Fatalf("Không tìm thấy dữ liệu cho chain %s", chainName)
	}

	chainData.Config.EthContractAddress = strings.ToLower(chainData.Config.EthContractAddress)
	chainData.Config.UsdcContractAddress = strings.ToLower(chainData.Config.UsdcContractAddress)
	chainData.Config.UsdtContractAddress = strings.ToLower(chainData.Config.UsdtContractAddress)
	chainData.Config.WrappedBTCAddress = strings.ToLower(chainData.Config.WrappedBTCAddress)

	addresses := []common.Address{
		common.HexToAddress(chainData.Config.EthContractAddress),
		common.HexToAddress(chainData.Config.UsdcContractAddress),
		common.HexToAddress(chainData.Config.UsdtContractAddress),
		common.HexToAddress(chainData.Config.WrappedBTCAddress),
	}


	// var topics [][]common.Hash
	// if chainData.Config.TransferSignature != "" {
	// 	topics = [][]common.Hash{
	// 		{common.HexToHash(chainData.Config.TransferSignature)},
	// 	}
	// }

	return ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Addresses: addresses,
		// Topics:    topics,
	}
}

func log_to_map(logData *types.Log, chainName string) map[string]interface{} {
	logMap := make(map[string]interface{})

	logMap["name_chain"] = chainName
	logMap["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	logMap["block_number"] = logData.BlockNumber
	logMap["tx_hash"] = logData.TxHash.Hex()
	logMap["address"] = logData.Address.Hex()
	logMap["log_index"] = logData.Index

	amount := new(big.Int).SetBytes(logData.Data)
	logMap["amount"] = amount.String()
	logMap["raw_data"] = fmt.Sprintf("%x", logData.Data)

	if len(logData.Topics) > 0 {
		eventSignature := logData.Topics[0].Hex()
		logMap["event_signature"] = eventSignature

		if transactionType, err := Parse_event_signature_name(eventSignature); err == nil {
			logMap["transaction_type"] = transactionType
		} else {
			log.Printf("Không thể parse event signature %s: %v", eventSignature, err)
			logMap["transaction_type"] = "Unknown"
		}

		for i, topic := range logData.Topics {
			logMap[fmt.Sprintf("topic_%d", i)] = topic.Hex()
		}
	}

	if len(logData.Topics) > 1 {
		fromAddr := common.HexToAddress(logData.Topics[1].Hex()).Hex()
		logMap["from_address"] = fromAddr
	}

	if len(logData.Topics) > 2 {
		toAddr := common.HexToAddress(logData.Topics[2].Hex()).Hex()
		logMap["to_address"] = toAddr
	}

	chainData := GetChainData(chainName)
	if chainData != nil {
		chainData.LogData = logMap
	} else {
		log.Printf("Cảnh báo: Không tìm thấy dữ liệu cho chain %s", chainName)
	}

	return logMap
}
