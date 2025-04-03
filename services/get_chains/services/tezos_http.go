package services

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"main/services/get_chains/configs"
	"main/services/get_chains/model"
)

// Khởi tạo dữ liệu cho chuỗi Tezos
func InitChainData(chainName string) *model.ChainDataTezos {
	if data, exists := model.ChainDataMapVan[chainName]; exists {
		return data.(*model.ChainDataTezos)
	}

	data := &model.ChainDataTezos{
		LastProcessedBlock: 0,
		LogData:            make(map[string]interface{}),
	}
	model.ChainDataMapVan[chainName] = data
	return data
}

// Lấy thông tin block từ Tezos blockchain qua RPC
func getTezosBlock(height int64, rpcURL string) (*model.BlockResponseVan, error) {
	url := fmt.Sprintf("%s/chains/main/blocks/%d", rpcURL, height)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error calling Tezos API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var blockResp model.BlockResponseVan
	if err := json.Unmarshal(body, &blockResp); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return &blockResp, nil
}

// Xử lý một block Tezos
func processTezosBlock(height int64, rpcURL string, chainName string) error {
	model.LogMutexVan.Lock()
	log.Printf("🔍 Fetching block info: %d", height)
	model.LogMutexVan.Unlock()

	block, err := getTezosBlock(height, rpcURL)
	if err != nil {
		model.LogMutexVan.Lock()
		log.Printf("❌ Error fetching block %d: %v", height, err)
		model.LogMutexVan.Unlock()
		return err
	}

	model.LogMutexVan.Lock()
	log.Printf("✅ Fetched block: %d", height)
	model.LogMutexVan.Unlock()

	var transactions []model.TransactionRecordVan
	txCount := 0
	for _, ops := range block.Operations {
		for _, op := range ops {
			for _, content := range op.Contents {
				if content.Kind == "transaction" {
					txCount++
					txHash := fmt.Sprintf("%x", sha256.Sum256([]byte(content.Source+content.Destination+content.Amount)))
					record := model.TransactionRecordVan{
						BlockHeight: fmt.Sprintf("%d", block.Header.Level),
						BlockHash:   block.Hash,
						BlockTime:   block.Header.Timestamp,
						ChainID:     block.ChainID,
						TxHash:      txHash,
						From:        content.Source,
						To:          content.Destination,
						Amount:      content.Amount,
						Token:       "XTZ",
						TotalAmount: content.Amount,
						TxType:      "transfer",
						Timestamp:   block.Header.Timestamp.Format(time.RFC3339),
					}
					transactions = append(transactions, record)

				}
			}
		}
	}

	model.LogMutexVan.Lock()
	log.Printf("BLOCK MỚI #%d | Chain: %s | Time: %s | Hash: %s",
		block.Header.Level, block.ChainID, block.Header.Timestamp.Format(time.RFC3339), block.Hash)
	model.LogMutexVan.Unlock()

	if txCount == 0 {
		model.LogMutexVan.Lock()
		log.Printf("Block #%d has no transactions", block.Header.Level)
		model.LogMutexVan.Unlock()
	} else {
		model.LogMutexVan.Lock()
		log.Printf("Number of transactions in block #%d: %d", block.Header.Level, txCount)
		model.LogMutexVan.Unlock()

		for _, record := range transactions {
			recordJSON, _ := json.MarshalIndent(record, "", "  ")
			model.LogMutexVan.Lock()
			log.Printf("GIAO DỊCH TRONG BLOCK #%d:\n%s", block.Header.Level, string(recordJSON))
			model.LogMutexVan.Unlock()
		}
	}

	model.LogMutexVan.Lock()
	log.Printf("✅ Completed processing block %d", height)
	model.LogMutexVan.Unlock()
	return nil
}

// Quét từ quá khứ đến hiện tại
func continueHandleTezosHTTP(chainName string) {
	chainDataGeneric := configs.GetChainData(chainName)
	if chainDataGeneric == nil {
		log.Fatalf("Data not found for chain %s", chainName)
	}
	chainData := chainDataGeneric.(*model.ChainDataTezos)

	startBlockNumber := int64(1)
	chainData.SetLastProcessedBlockVan(startBlockNumber - 1)

	model.LogMutexVan.Lock()
	log.Printf("======= STARTING SYSTEM FOR TEZOS CHAIN %s =======", chainName)
	log.Printf("Initialized lastProcessedBlock = %d", chainData.GetLastProcessedBlockVan())
	log.Printf("Starting scan from block = %d", startBlockNumber)
	log.Printf("==================================")
	model.LogMutexVan.Unlock()

	blockCounter := 0
	currentBlock := startBlockNumber

	for {
		blockCounter++

		if blockCounter%50 == 0 {
			url := fmt.Sprintf("%s/chains/main/blocks/head/header", chainData.Config.RPC)
			resp, err := http.Get(url)
			if err == nil {
				body, err := io.ReadAll(resp.Body)
				if err == nil {
					var header struct {
						Level int64 `json:"level"`
					}
					if json.Unmarshal(body, &header) == nil {
						latestBlock := header.Level
						gap := latestBlock - currentBlock
						if gap > 20 {
							nextBatchEnd := currentBlock + 100
							if nextBatchEnd > latestBlock {
								nextBatchEnd = latestBlock
							}

							model.LogMutexVan.Lock()
							log.Printf("⚠️ Detected gap of %d blocks on chain %s. Fast scanning from %d to %d...",
								gap, chainName, currentBlock, nextBatchEnd)
							model.LogMutexVan.Unlock()

							for i := currentBlock; i <= nextBatchEnd; i++ {
								if err := processTezosBlock(i, chainData.Config.RPC, chainName); err != nil {
									model.LogMutexVan.Lock()
									log.Printf("Error processing block %d: %v, continuing...", i, err)
									model.LogMutexVan.Unlock()
								}
								time.Sleep(50 * time.Millisecond)
							}

							currentBlock = nextBatchEnd + 1
							model.ProcessLockVan.Lock()
							chainData.SetLastProcessedBlockVan(nextBatchEnd)
							model.ProcessLockVan.Unlock()

							model.LogMutexVan.Lock()
							log.Printf("✅ Fast scanned to block %d", nextBatchEnd)
							model.LogMutexVan.Unlock()
							continue
						}
					}
				}
				resp.Body.Close()
			}
		}

		err := processTezosBlock(currentBlock, chainData.Config.RPC, chainName)
		if err == nil {
			currentBlock++
			model.ProcessLockVan.Lock()
			if currentBlock-1 > chainData.GetLastProcessedBlockVan() {
				chainData.SetLastProcessedBlockVan(currentBlock - 1)
			}
			model.ProcessLockVan.Unlock()
			time.Sleep(200 * time.Millisecond)
		} else {
			sleepTime := 500 * time.Millisecond
			model.LogMutexVan.Lock()
			log.Printf("Waiting %v before retrying block %d", sleepTime, currentBlock)
			model.LogMutexVan.Unlock()
			time.Sleep(sleepTime)
		}
	}
}

// Quét từ hiện tại ngược về quá khứ
func reverseHandleTezosHTTP(chainName string, pastDuration time.Duration) {
	chainDataGeneric := configs.GetChainData(chainName)
	if chainDataGeneric == nil {
		log.Fatalf("Data not found for chain %s", chainName)
	}
	chainData := chainDataGeneric.(*model.ChainDataTezos)

	url := fmt.Sprintf("%s/chains/main/blocks/head/header", chainData.Config.RPC)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error fetching chain status: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	var header struct {
		Level     int64     `json:"level"`
		Timestamp time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(body, &header); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	latestBlock := header.Level
	latestBlockTime := header.Timestamp
	targetTime := latestBlockTime.Add(-pastDuration)

	model.LogMutexVan.Lock()
	log.Printf("======= STARTING REVERSE SYSTEM FOR TEZOS CHAIN %s =======", chainName)
	log.Printf("Starting scan from current block = %d", latestBlock)
	log.Printf("Target time: %v", targetTime)
	log.Printf("==================================")
	model.LogMutexVan.Unlock()

	blockCounter := 0
	currentBlock := latestBlock

	for {
		blockCounter++

		err := processTezosBlock(currentBlock, chainData.Config.RPC, chainName)
		if err == nil {
			block, err := getTezosBlock(currentBlock, chainData.Config.RPC)
			if err == nil {
				blockTime := block.Header.Timestamp
				if blockTime.Before(targetTime) || blockTime.Equal(targetTime) {
					model.LogMutexVan.Lock()
					log.Printf("✅ Reached target time at block %d", currentBlock)
					model.LogMutexVan.Unlock()
					break
				}
			}

			currentBlock--
			model.ProcessLockVan.Lock()
			chainData.SetLastProcessedBlockVan(currentBlock + 1)
			model.ProcessLockVan.Unlock()

			if blockCounter%50 == 0 {
				model.LogMutexVan.Lock()
				log.Printf("🔄 Scanned %d blocks backwards, currently at block %d", blockCounter, currentBlock)
				model.LogMutexVan.Unlock()
			}

			time.Sleep(200 * time.Millisecond)
		} else {
			sleepTime := 500 * time.Millisecond
			model.LogMutexVan.Lock()
			log.Printf("Waiting %v before retrying block %d", sleepTime, currentBlock)
			model.LogMutexVan.Unlock()
			time.Sleep(sleepTime)
		}

		if currentBlock <= 1 {
			model.LogMutexVan.Lock()
			log.Printf("✅ Reached the first block")
			model.LogMutexVan.Unlock()
			break
		}
	}

	model.LogMutexVan.Lock()
	log.Printf("✅ Completed processing blocks backwards for chain %s", chainName)
	model.LogMutexVan.Unlock()
}

// Hàm chính xử lý Tezos qua HTTP
func Handle_tezos_http() {
	chainName := "tezos"

	// Tạo thư mục log nếu chưa tồn tại
	if _, err := os.Stat("./log"); os.IsNotExist(err) {
		os.Mkdir("./log", 0755)
	}

	logFile, err := os.OpenFile("./log/tezos_http.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Cannot open log file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	chainData := InitChainData(chainName)
	log.Printf("Initialized data for chain %s, last processed block: %d",
		chainName, chainData.GetLastProcessedBlockVan())

	if err := configs.LoadConfig("./services/get_chains/configs/config-tezos.json", chainName); err != nil {
		log.Fatalf("Cannot load config: %v", err)
	}

	pastDuration := 1 * time.Hour

	model.LogMutexVan.Lock()
	log.Printf("======= BEGIN PROCESSING TEZOS CHAIN %s =======", chainName)
	log.Printf("Step 1: Scanning backwards into the past (%v)", pastDuration)
	model.LogMutexVan.Unlock()

	reverseHandleTezosHTTP(chainName, pastDuration)

	model.LogMutexVan.Lock()
	log.Printf("Step 2: Continuing scan from past to present and monitoring new blocks")
	model.LogMutexVan.Unlock()

	continueHandleTezosHTTP(chainName)
}
