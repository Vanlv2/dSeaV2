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

// Khởi tạo dữ liệu cho chuỗi MultiversX
func InitElrondChainData(chainName string) *model.ChainDataElrond {
	if data, exists := model.ChainDataMapVan[chainName]; exists {
		return data.(*model.ChainDataElrond)
	}

	data := &model.ChainDataElrond{
		LastProcessedBlock: 0,
		LogData:            make(map[string]interface{}),
	}
	model.ChainDataMapVan[chainName] = data
	return data
}

// Lấy thông tin block từ MultiversX blockchain qua API
func getElrondBlock(nonce int64, apiURL string) (*model.ElrondBlockResponseVan, error) {
	url := fmt.Sprintf("%s/blocks?nonces=%d", apiURL, nonce)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error calling MultiversX API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// Debug: Log dữ liệu thô
	log.Printf("Raw block response for nonce %d: %s", nonce, string(body))

	var blockResp []model.ElrondBlockResponseVan
	if err := json.Unmarshal(body, &blockResp); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	if len(blockResp) == 0 {
		return nil, fmt.Errorf("no block found for nonce %d", nonce)
	}

	return &blockResp[0], nil
}

// Lấy giao dịch từ block
func getElrondTransactions(blockHash string, apiURL string) ([]model.ElrondTransactionVan, error) {
	url := fmt.Sprintf("%s/blocks/%s/transactions", apiURL, blockHash)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching transactions: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading transactions response: %v", err)
	}

	// Debug: Log dữ liệu thô
	log.Printf("Raw transactions response for block hash %s: %s", blockHash, string(body))

	var txResponse struct {
		Data struct {
			Transactions []model.ElrondTransactionVan `json:"transactions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &txResponse); err != nil {
		return nil, fmt.Errorf("error parsing transactions JSON: %v", err)
	}

	return txResponse.Data.Transactions, nil
}

// Xử lý một block MultiversX
func processElrondBlock(nonce int64, apiURL string) error {
	model.LogMutexVan.Lock()
	log.Printf("🔍 Fetching block info: %d", nonce)
	model.LogMutexVan.Unlock()

	block, err := getElrondBlock(nonce, apiURL)
	if err != nil {
		model.LogMutexVan.Lock()
		log.Printf("❌ Error fetching block %d: %v", nonce, err)
		model.LogMutexVan.Unlock()
		return err
	}

	model.LogMutexVan.Lock()
	log.Printf("✅ Fetched block: %d", nonce)
	model.LogMutexVan.Unlock()

	// Lấy giao dịch từ block
	transactions, err := getElrondTransactions(block.Hash, apiURL)
	if err != nil {
		model.LogMutexVan.Lock()
		log.Printf("❌ Error fetching transactions for block %d: %v", nonce, err)
		model.LogMutexVan.Unlock()
		return err
	}

	var txRecords []model.TransactionRecordVan
	for _, tx := range transactions {
		txHash := tx.Hash
		if txHash == "" {
			txHash = fmt.Sprintf("%x", sha256.Sum256([]byte(tx.Sender+tx.Receiver+fmt.Sprintf("%d", tx.Value))))
		}
		blockTime := time.Unix(block.Timestamp, 0).UTC()
		record := model.TransactionRecordVan{
			BlockHeight: fmt.Sprintf("%d", block.Nonce),
			BlockHash:   block.Hash,
			BlockTime:   blockTime,
			ChainID:     "elrond-mainnet",
			TxHash:      txHash,
			From:        tx.Sender,
			To:          tx.Receiver,
			Amount:      fmt.Sprintf("%d", tx.Value),
			Token:       "EGLD",
			TotalAmount: fmt.Sprintf("%d", tx.Value),
			TxType:      "transfer",
			Timestamp:   blockTime.Format(time.RFC3339),
		}
		txRecords = append(txRecords, record)
	}

	txCount := len(txRecords)

	if txCount == 0 {
		model.LogMutexVan.Lock()
		log.Printf("Block #%d has no transactions", block.Nonce)
		model.LogMutexVan.Unlock()
	} else {
		for _, record := range txRecords {
			recordJSON, _ := json.MarshalIndent(record, "", "  ")
			model.LogMutexVan.Lock()
			log.Printf("GIAO DỊCH TRONG BLOCK #%d:\n%s", block.Nonce, string(recordJSON))
			model.LogMutexVan.Unlock()
		}
	}

	model.LogMutexVan.Lock()
	log.Printf("✅ Completed processing block %d", nonce)
	model.LogMutexVan.Unlock()
	return nil
}

// Quét từ quá khứ đến hiện tại
func continueHandleElrondHTTP(chainName string) {
	chainDataGeneric := configs.GetChainData(chainName)
	if chainDataGeneric == nil {
		log.Fatalf("Data not found for chain %s", chainName)
	}
	chainData := chainDataGeneric.(*model.ChainDataElrond)

	startBlockNumber := int64(1)
	chainData.SetLastProcessedBlockVan(startBlockNumber - 1)

	model.LogMutexVan.Lock()
	log.Printf("======= STARTING SYSTEM FOR MULTIVERSX CHAIN %s =======", chainName)
	log.Printf("Initialized lastProcessedBlock = %d", chainData.GetLastProcessedBlockVan())
	log.Printf("Starting scan from block = %d", startBlockNumber)
	log.Printf("==================================")
	model.LogMutexVan.Unlock()

	blockCounter := 0
	currentBlock := startBlockNumber

	for {
		blockCounter++

		if blockCounter%50 == 0 {
			url := fmt.Sprintf("%s/network/status/4294967295", chainData.Config.API)
			resp, err := http.Get(url)
			if err == nil {
				body, err := io.ReadAll(resp.Body)
				if err == nil {
					var status struct {
						Data struct {
							Status struct {
								HighestFinalNonce int64 `json:"erd_highest_final_nonce"`
							} `json:"status"`
						} `json:"data"`
					}
					if json.Unmarshal(body, &status) == nil {
						latestBlock := status.Data.Status.HighestFinalNonce
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
								if err := processElrondBlock(i, chainData.Config.API); err != nil {
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
			}
		}

		err := processElrondBlock(currentBlock, chainData.Config.API)
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
func reverseHandleElrondHTTP(chainName string, pastDuration time.Duration) {
	chainDataGeneric := configs.GetChainData(chainName)
	if chainDataGeneric == nil {
		log.Fatalf("Data not found for chain %s", chainName)
	}
	chainData := chainDataGeneric.(*model.ChainDataElrond)

	url := fmt.Sprintf("%s/network/status/4294967295", chainData.Config.API)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error fetching chain status: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	var status struct {
		Data struct {
			Status struct {
				HighestFinalNonce int64 `json:"erd_highest_final_nonce"`
			} `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &status); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	latestBlock := status.Data.Status.HighestFinalNonce
	latestBlockInfo, err := getElrondBlock(latestBlock, chainData.Config.API)
	if err != nil {
		log.Fatalf("Error fetching latest block: %v", err)
	}
	latestBlockTime := time.Unix(latestBlockInfo.Timestamp, 0).UTC()

	targetTime := latestBlockTime.Add(-pastDuration)

	model.LogMutexVan.Lock()
	log.Printf("======= STARTING REVERSE SYSTEM FOR MULTIVERSX CHAIN %s =======", chainName)
	log.Printf("Starting scan from current block = %d", latestBlock)
	log.Printf("Target time: %v", targetTime)
	log.Printf("==================================")
	model.LogMutexVan.Unlock()

	blockCounter := 0
	currentBlock := latestBlock

	for {
		blockCounter++

		err := processElrondBlock(currentBlock, chainData.Config.API)
		if err == nil {
			block, err := getElrondBlock(currentBlock, chainData.Config.API)
			if err == nil {
				blockTime := time.Unix(block.Timestamp, 0).UTC()
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

// Hàm chính xử lý MultiversX qua HTTP
func Handle_elrond_http() {
	chainName := "elrond"
	logFile, err := os.OpenFile("./log/elrond_http.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Cannot open log file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	chainData := InitElrondChainData(chainName)
	log.Printf("Initialized data for chain %s, last processed block: %d",
		chainName, chainData.GetLastProcessedBlockVan())

	if err := configs.LoadConfig("./configs/config-elrond.json", chainName); err != nil {
		log.Fatalf("Cannot load config: %v", err)
	}

	pastDuration := 1 * time.Hour

	model.LogMutexVan.Lock()
	log.Printf("======= BEGIN PROCESSING MULTIVERSX CHAIN %s =======", chainName)
	log.Printf("Step 1: Scanning backwards into the past (%v)", pastDuration)
	model.LogMutexVan.Unlock()

	reverseHandleElrondHTTP(chainName, pastDuration)

	model.LogMutexVan.Lock()
	log.Printf("Step 2: Continuing scan from past to present and monitoring new blocks")
	model.LogMutexVan.Unlock()

	continueHandleElrondHTTP(chainName)
}
