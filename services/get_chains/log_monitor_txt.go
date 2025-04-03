package main

import (
	"context"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

func periodic_logs_monitoring(ctx context.Context, client *ethclient.Client, chainName string) {
	chainData := GetChainData(chainName)
	if chainData == nil {
		log.Printf("Không tìm thấy dữ liệu cho chain %s", chainName)
		return
	}

	ticker := time.NewTicker(time.Duration(chainData.Config.TimeNeedToBlock) * time.Millisecond)
	defer ticker.Stop()

	log.Printf("Bắt đầu giám sát các logs bị bỏ lỡ cho chain %s (khoảng thời gian: %d ms)",
		chainName, chainData.Config.TimeNeedToBlock)

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
				log.Printf("Không thể lấy số khối mới nhất cho chain %s: %v", chainName, err)
				continue
			}

			processLock.Lock()
			currentLastProcessed := new(big.Int).Set(chainData.LastProcessedBlock)
			processLock.Unlock()

			currentLatestBlock := big.NewInt(int64(latestBlock))
			blockDiff := new(big.Int).Sub(currentLatestBlock, currentLastProcessed)

			if blockDiff.Cmp(big.NewInt(1)) > 0 {
				log.Printf("Phát hiện %d khối bị bỏ lỡ cho chain %s. Đang lấy logs...",
					blockDiff, chainName)

				select {
				case chainData.DisconnectedChannel <- struct{}{}:
				default:
					log.Printf("Channel DisconnectedChannel đã đầy, bỏ qua tín hiệu lấy logs bị bỏ lỡ")
				}
			}

		case <-ctx.Done():
			log.Println("Dừng giám sát logs bị bỏ lỡ theo yêu cầu context")
			return
		}
	}
}

func handle_disconnected_logs(client *ethclient.Client, chainName string) {
	chainData := GetChainData(chainName)
	if chainData == nil {
		log.Printf("Không tìm thấy dữ liệu cho chain %s", chainName)
		return
	}

	processLock.Lock()
	if chainData.IsProcessingReorg {
		log.Printf("Đã có một quá trình xử lý reorg đang chạy cho chain %s, bỏ qua...", chainName)
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
		log.Printf("❌ Không thể lấy số khối mới nhất cho chain %s: %v", chainName, err)
		return
	}

	processLock.Lock()
	fromBlock := new(big.Int).Add(chainData.LastProcessedBlock, big.NewInt(1))
	processLock.Unlock()

	toBlock := big.NewInt(int64(latestBlock))

	if fromBlock.Cmp(toBlock) > 0 {
		log.Printf("ℹ️ Không có khối mới để xử lý cho chain %s. Khối mới nhất đã xử lý: %d",
			chainName, chainData.LastProcessedBlock)
		return
	}

	if blockDiff := new(big.Int).Sub(toBlock, fromBlock).Int64(); blockDiff > 5000 {
		log.Printf("⚠️ Khoảng khối quá lớn (%d) cho chain %s, giới hạn xuống 5000 khối", blockDiff, chainName)
		toBlock = new(big.Int).Add(fromBlock, big.NewInt(5000))
	}

	log.Printf("🔍 Đang lấy các log bị bỏ lỡ từ khối %d đến %d cho chain %s...",
		fromBlock, toBlock, chainName)

	process_query_range(client, chainName, fromBlock, toBlock)

	processLock.Lock()
	chainData.LastProcessedBlock = toBlock
	processLock.Unlock()

	log.Printf("✅ Đã cập nhật khối cuối cùng được xử lý cho %s = %d", chainName, toBlock)
	log.Printf("🎉 Đã xử lý xong tất cả logs bị bỏ lỡ trong khoảng từ %d đến %d cho chain %s",
		fromBlock, toBlock, chainName)
}

func process_query_range(client *ethclient.Client, chainName string, fromBlock, toBlock *big.Int) {
	maxBlockRange := big.NewInt(1000)
	blockDiff := new(big.Int).Sub(toBlock, fromBlock)
	totalBlocks := blockDiff.Int64() + 1

	log.Printf("📊 Bắt đầu quét %d khối từ %d đến %d cho chain %s",
		totalBlocks, fromBlock, toBlock, chainName)

	if blockDiff.Cmp(maxBlockRange) > 0 {
		log.Printf("⚙️ Khoảng khối quá lớn (%d), đang chia thành các khoảng nhỏ hơn...", totalBlocks)

		currentFromBlock := new(big.Int).Set(fromBlock)
		processedRanges := 0
		totalRanges := (blockDiff.Int64() / maxBlockRange.Int64()) + 1
		totalLogs := 0

		for currentFromBlock.Cmp(toBlock) < 0 {
			processedRanges++

			currentToBlock := new(big.Int).Add(currentFromBlock, maxBlockRange)
			if currentToBlock.Cmp(toBlock) > 0 {
				currentToBlock.Set(toBlock)
			}

			rangeSize := new(big.Int).Sub(currentToBlock, currentFromBlock).Int64() + 1

			log.Printf("🔄 Xử lý khoảng con (%d/%d): %d đến %d (%d khối) cho chain %s",
				processedRanges, totalRanges, currentFromBlock, currentToBlock, rangeSize, chainName)

			query := create_query(chainName, currentFromBlock, currentToBlock)

			queryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			logs, err := client.FilterLogs(queryCtx, query)
			cancel()

			if err != nil {
				log.Printf("❌ Lỗi khi lấy logs từ khối %d đến %d cho chain %s: %v",
					currentFromBlock, currentToBlock, chainName, err)
			} else {
				logCount := len(logs)
				totalLogs += logCount

				if logCount > 0 {
					log.Printf("✅ Tìm thấy %d giao dịch trong khoảng từ %d đến %d cho chain %s",
						logCount, currentFromBlock, currentToBlock, chainName)

					for i, vLog := range logs {
						log.Printf("   📝 Xử lý giao dịch (%d/%d) tại khối %d, tx %s",
							i+1, logCount, vLog.BlockNumber, vLog.TxHash.Hex())
						process_handle_log(vLog, chainName)
					}
				} else {
					log.Printf("⚠️ Không tìm thấy giao dịch nào trong khoảng từ %d đến %d cho chain %s",
						currentFromBlock, currentToBlock, chainName)
				}
			}

			currentFromBlock = new(big.Int).Add(currentToBlock, big.NewInt(1))
		}

		log.Printf("📋 Tổng kết: Đã quét %d khoảng, tìm thấy %d giao dịch từ khối %d đến %d cho chain %s",
			processedRanges, totalLogs, fromBlock, toBlock, chainName)
	} else {
		query := create_query(chainName, fromBlock, toBlock)

		queryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		logs, err := client.FilterLogs(queryCtx, query)
		cancel()

		if err != nil {
			log.Printf("❌ Lỗi khi lấy logs từ khối %d đến %d cho chain %s: %v",
				fromBlock, toBlock, chainName, err)
			return
		}

		logCount := len(logs)
		if logCount > 0 {
			log.Printf("✅ Tìm thấy %d giao dịch trong khoảng từ khối %d đến %d cho chain %s",
				logCount, fromBlock, toBlock, chainName)

			for i, vLog := range logs {
				log.Printf("   📝 Xử lý giao dịch (%d/%d) tại khối %d, tx %s",
					i+1, logCount, vLog.BlockNumber, vLog.TxHash.Hex())
				process_handle_log(vLog, chainName)
			}
		} else {
			log.Printf("⚠️ Không tìm thấy giao dịch nào trong khoảng từ khối %d đến %d cho chain %s",
				fromBlock, toBlock, chainName)
		}
	}
}

func clean_processed_transactions(chainName string, cutoffBlock uint64) {
	chainData := GetChainData(chainName)
	if chainData == nil {
		return
	}

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
	log.Printf("✅ Đã làm sạch map processedTransactions cho %s. Kích thước mới: %d",
		chainName, len(chainData.ProcessedTxs))
}
