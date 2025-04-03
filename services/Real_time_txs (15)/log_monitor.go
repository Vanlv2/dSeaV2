package main

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

func periodic_logs_monitoring(ctx context.Context, client *ethclient.Client, chainName string) {
	chainData := GetChainData(chainName)
	if chainData == nil {
		return
	}

	ticker := time.NewTicker(time.Duration(chainData.Config.TimeNeedToBlock) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if ctx.Err() != nil {
				return
			}

			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			latestBlock, err := client.BlockNumber(bgCtx)
			cancel()

			if err != nil {
				fmt.Printf("Không thể lấy số khối mới nhất cho chain %s: %v\n", chainName, err)
				continue
			}

			processLock.Lock()
			// Nếu LastProcessedBlock vẫn là 0, cập nhật nó thành khối mới nhất
			if chainData.LastProcessedBlock.Cmp(big.NewInt(0)) == 0 {
				chainData.LastProcessedBlock = big.NewInt(int64(latestBlock))
				processLock.Unlock()
				continue
			}

			currentLastProcessed := new(big.Int).Set(chainData.LastProcessedBlock)
			processLock.Unlock()

			currentLatestBlock := big.NewInt(int64(latestBlock))
			blockDiff := new(big.Int).Sub(currentLatestBlock, currentLastProcessed)

			if blockDiff.Cmp(big.NewInt(1)) > 0 {
				fmt.Printf("Phát hiện %d khối bị bỏ lỡ cho chain %s. Đang lấy logs...\n",
					blockDiff, chainName)

				select {
				case chainData.DisconnectedChannel <- struct{}{}:
				default:
					fmt.Printf("Channel DisconnectedChannel đã đầy, bỏ qua tín hiệu lấy logs bị bỏ lỡ\n")
				}
			}

		case <-ctx.Done():
			fmt.Println("Dừng giám sát logs bị bỏ lỡ theo yêu cầu context")
			return
		}
	}
}

func handle_disconnected_logs(client *ethclient.Client, chainName string) {
	chainData := GetChainData(chainName)
	if chainData == nil {
		return
	}

	processLock.Lock()
	if chainData.IsProcessingReorg {
		processLock.Unlock()
		return
	}
	chainData.IsProcessingReorg = true
	processLock.Unlock()

	defer func() {
		processLock.Lock()
		chainData.IsProcessingReorg = false
		processLock.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	latestBlock, err := client.BlockNumber(ctx)
	if err != nil {
		fmt.Printf("❌ Không thể lấy số khối mới nhất cho chain %s: %v\n", chainName, err)
		return
	}

	processLock.Lock()
	// Nếu LastProcessedBlock vẫn là 0, cập nhật nó thành khối mới nhất
	if chainData.LastProcessedBlock.Cmp(big.NewInt(0)) == 0 {
		chainData.LastProcessedBlock = big.NewInt(int64(latestBlock))
		processLock.Unlock()
		return
	}

	fromBlock := new(big.Int).Add(chainData.LastProcessedBlock, big.NewInt(1))
	processLock.Unlock()

	toBlock := big.NewInt(int64(latestBlock))

	if fromBlock.Cmp(toBlock) > 0 {
		fmt.Printf("ℹ️ Không có khối mới để xử lý cho chain %s. Khối mới nhất đã xử lý: %d\n",
			chainName, chainData.LastProcessedBlock)
		return
	}

	// Giảm kích thước khoảng tối đa xuống 1000 khối
	if blockDiff := new(big.Int).Sub(toBlock, fromBlock).Int64(); blockDiff > 1000 {
		fmt.Printf("⚠️ Khoảng khối quá lớn (%d) cho chain %s, giới hạn xuống 1000 khối\n", blockDiff, chainName)
		toBlock = new(big.Int).Add(fromBlock, big.NewInt(1000))
	}

	process_query_range(client, chainName, fromBlock, toBlock)

	processLock.Lock()
	chainData.LastProcessedBlock = toBlock
	processLock.Unlock()

}

func process_query_range(client *ethclient.Client, chainName string, fromBlock, toBlock *big.Int) {
	// Giảm kích thước khoảng khối tối đa xuống để tránh lỗi API
	maxBlockRange := big.NewInt(100) // Giảm từ 1000 xuống 100
	blockDiff := new(big.Int).Sub(toBlock, fromBlock)

	if blockDiff.Cmp(maxBlockRange) > 0 {
		currentFromBlock := new(big.Int).Set(fromBlock)
		processedRanges := 0
		totalLogs := 0

		for currentFromBlock.Cmp(toBlock) < 0 {
			processedRanges++

			currentToBlock := new(big.Int).Add(currentFromBlock, maxBlockRange)
			if currentToBlock.Cmp(toBlock) > 0 {
				currentToBlock.Set(toBlock)
			}
			query := create_query(chainName, currentFromBlock, currentToBlock)

			queryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			logs, err := client.FilterLogs(queryCtx, query)
			cancel()

			if err != nil {
				fmt.Printf("❌ Lỗi khi lấy logs từ khối %d đến %d cho chain %s: %v\n",
					currentFromBlock, currentToBlock, chainName, err)

				// Nếu gặp lỗi Block range limit exceeded, giảm kích thước khoảng xuống một nửa và thử lại
				if strings.Contains(err.Error(), "Block range limit exceeded") {
					fmt.Printf("⚠️ Giảm kích thước khoảng xuống một nửa và thử lại\n")

					// Giảm kích thước khoảng xuống một nửa
					halfRange := new(big.Int).Div(new(big.Int).Sub(currentToBlock, currentFromBlock), big.NewInt(2))
					if halfRange.Cmp(big.NewInt(0)) > 0 {
						processedRanges--
						continue
					}
				}
			} else {
				logCount := len(logs)
				totalLogs += logCount

				if logCount > 0 {
					for _, vLog := range logs {
						process_handle_log(vLog, chainName)
					}
				} else {
					fmt.Printf("⚠️ Không tìm thấy giao dịch nào trong khoảng từ %d đến %d cho chain %s\n",
						currentFromBlock, currentToBlock, chainName)
				}
			}

			currentFromBlock = new(big.Int).Add(currentToBlock, big.NewInt(1))
		}

	} else {
		query := create_query(chainName, fromBlock, toBlock)

		queryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		logs, err := client.FilterLogs(queryCtx, query)
		cancel()

		if err != nil {
			fmt.Printf("❌ Lỗi khi lấy logs từ khối %d đến %d cho chain %s: %v\n",
				fromBlock, toBlock, chainName, err)
			return
		}

		logCount := len(logs)
		if logCount > 0 {
			for _, vLog := range logs {
				process_handle_log(vLog, chainName)
			}
		} else {
			fmt.Printf("⚠️ Không tìm thấy giao dịch nào trong khoảng từ khối %d đến %d cho chain %s\n",
				fromBlock, toBlock, chainName)
		}
	}
}

func clean_processed_transactions(chainName string, cutoffBlock uint64) {
	chainData := GetChainData(chainName)
	if chainData == nil {
		return
	}

	oldSize := len(chainData.ProcessedTxs)
	newMap := make(map[string]bool)
	for k, v := range chainData.ProcessedTxs {
		parts := strings.Split(k, "-")
		if len(parts) > 0 {
			blockNum, _ := strconv.ParseUint(parts[0], 10, 64)
			if blockNum >= cutoffBlock {
				newMap[k] = v
			}
		}
	}
	chainData.ProcessedTxs = newMap
	fmt.Printf("✅ Đã làm sạch map processedTransactions cho %s. Kích thước cũ: %d, kích thước mới: %d\n",
		chainName, oldSize, len(chainData.ProcessedTxs))
}
