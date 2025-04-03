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

// Khởi tạo dữ liệu cho chuỗi Stellar
func InitChainDataStellar(chainName string) *model.ChainDataStellarVan {
	if data, exists := model.ChainDataMapVan[chainName]; exists {
		return data.(*model.ChainDataStellarVan)
	}

	data := &model.ChainDataStellarVan{
		LastProcessedLedger: 0,
		LogData:             make(map[string]interface{}),
	}
	model.ChainDataMapVan[chainName] = data
	return data
}

// Lấy thông tin ledger từ Stellar Horizon API
func getStellarLedger(ledger int64, horizonURL string) (*model.StellarLedgerResponseVan, error) {
	url := fmt.Sprintf("%s/ledgers/%d", horizonURL, ledger)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error calling Stellar Horizon API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var ledgerResp model.StellarLedgerResponseVan
	if err := json.Unmarshal(body, &ledgerResp); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return &ledgerResp, nil
}

// Lấy danh sách giao dịch trong ledger từ Stellar Horizon API
func getStellarTransactions(ledger int64, horizonURL string) ([]model.StellarTransactionVan, error) {
	url := fmt.Sprintf("%s/ledgers/%d/transactions", horizonURL, ledger)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching transactions: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var txResp struct {
		Embedded struct {
			Records []model.StellarTransactionVan `json:"records"`
		} `json:"_embedded"`
	}
	if err := json.Unmarshal(body, &txResp); err != nil {
		return nil, fmt.Errorf("error parsing transactions JSON: %v", err)
	}

	return txResp.Embedded.Records, nil
}

// Xử lý một ledger Stellar
func processStellarLedger(ledger int64, horizonURL string) error {
	model.LogMutexVan.Lock()
	log.Printf("🔍 Fetching ledger info: %d", ledger)
	model.LogMutexVan.Unlock()

	ledgerData, err := getStellarLedger(ledger, horizonURL)
	if err != nil {
		model.LogMutexVan.Lock()
		log.Printf("❌ Error fetching ledger %d: %v", ledger, err)
		model.LogMutexVan.Unlock()
		return err
	}

	model.LogMutexVan.Lock()
	log.Printf("✅ Fetched ledger: %d", ledger)
	model.LogMutexVan.Unlock()

	transactions, err := getStellarTransactions(ledger, horizonURL)
	if err != nil {
		model.LogMutexVan.Lock()
		log.Printf("❌ Error fetching transactions for ledger %d: %v", ledger, err)
		model.LogMutexVan.Unlock()
		return err
	}

	var txEvents []model.TransactionEventVan
	txCount := len(transactions)

	// Nếu không có giao dịch, tạo một event cho ledger
	if txCount == 0 {
		event := model.TransactionEventVan{
			Address:         "", // Không có địa chỉ cụ thể cho ledger
			Amount:          "0",
			BlockNumber:     ledgerData.Sequence,
			EventSignature:  "",
			FromAddress:     "",
			LogIndex:        0,
			NameChain:       "stellar",
			RawData:         fmt.Sprintf("%v", ledgerData),
			Timestamp:       ledgerData.Timestamp.Format("2006-01-02 15:04:05"),
			ToAddress:       "",
			TransactionType: "LedgerClose",
			TxHash:          ledgerData.Hash,
		}
		txEvents = append(txEvents, event)
	} else {
		// Xử lý từng giao dịch trong ledger
		for _, tx := range transactions {
			txHash := fmt.Sprintf("%x", sha256.Sum256([]byte(tx.Source+tx.Destination+tx.Amount)))
			event := model.TransactionEventVan{
				Address:         tx.Destination, // Địa chỉ nhận (có thể thay đổi tùy logic)
				Amount:          tx.Amount,
				BlockNumber:     ledgerData.Sequence,
				EventSignature:  "", // Không áp dụng cho Stellar
				FromAddress:     tx.Source,
				LogIndex:        0, // Stellar không có log index, giữ 0
				NameChain:       "stellar",
				RawData:         fmt.Sprintf("%v", tx),
				Timestamp:       tx.Timestamp.Format("2006-01-02 15:04:05"),
				ToAddress:       tx.Destination,
				TransactionType: "Transfer", // Có thể thay đổi dựa trên tx.AssetType
				TxHash:          txHash,
			}
			txEvents = append(txEvents, event)
		}
	}

	// Ghi log thông tin ledger
	model.LogMutexVan.Lock()
	log.Printf("LEDGER MỚI #%d | Chain: %s | Time: %s | Hash: %s",
		ledgerData.Sequence, "stellar", ledgerData.Timestamp.Format("2006-01-02 15:04:05"), ledgerData.Hash)
	model.LogMutexVan.Unlock()

	if txCount == 0 {
		model.LogMutexVan.Lock()
		log.Printf("Ledger #%d has no transactions", ledgerData.Sequence)
		model.LogMutexVan.Unlock()
	} else {
		model.LogMutexVan.Lock()
		log.Printf("Number of transactions in ledger #%d: %d", ledgerData.Sequence, txCount)
		model.LogMutexVan.Unlock()
	}

	// Ghi log từng event
	for _, event := range txEvents {
		eventJSON, _ := json.MarshalIndent(event, "", "  ")
		model.LogMutexVan.Lock()
		log.Printf("EVENT TRONG LEDGER #%d:\n%s", ledgerData.Sequence, string(eventJSON))
		model.LogMutexVan.Unlock()
	}

	model.LogMutexVan.Lock()
	log.Printf("✅ Completed processing ledger %d", ledger)
	model.LogMutexVan.Unlock()
	return nil
}

// Quét từ quá khứ đến hiện tại
func continueHandleStellarHTTP(chainName string) {
	chainDataGeneric := configs.GetChainData(chainName)
	if chainDataGeneric == nil {
		log.Fatalf("Data not found for chain %s", chainName)
	}
	chainData := chainDataGeneric.GetConfigVan().(*model.ChainDataStellarVan)

	startLedgerNumber := int64(1)
	chainData.SetLastProcessedBlockVan(startLedgerNumber - 1)

	model.LogMutexVan.Lock()
	log.Printf("======= STARTING SYSTEM FOR STELLAR CHAIN %s =======", chainName)
	log.Printf("Initialized lastProcessedLedger = %d", chainData.GetLastProcessedBlockVan())
	log.Printf("Starting scan from ledger = %d", startLedgerNumber)
	log.Printf("==================================")
	model.LogMutexVan.Unlock()

	ledgerCounter := 0
	currentLedger := startLedgerNumber

	for {
		ledgerCounter++

		if ledgerCounter%50 == 0 {
			url := fmt.Sprintf("%s/ledgers?order=desc&limit=1", chainData.Config.HorizonURL)
			resp, err := http.Get(url)
			if err == nil {
				body, err := io.ReadAll(resp.Body)
				if err == nil {
					var latestResp struct {
						Embedded struct {
							Records []model.StellarLedgerResponseVan `json:"records"`
						} `json:"_embedded"`
					}
					if json.Unmarshal(body, &latestResp) == nil && len(latestResp.Embedded.Records) > 0 {
						latestLedger := latestResp.Embedded.Records[0].Sequence
						gap := latestLedger - currentLedger
						if gap > 20 {
							nextBatchEnd := currentLedger + 100
							if nextBatchEnd > latestLedger {
								nextBatchEnd = latestLedger
							}

							model.LogMutexVan.Lock()
							log.Printf("⚠️ Detected gap of %d ledgers on chain %s. Fast scanning from %d to %d...",
								gap, chainName, currentLedger, nextBatchEnd)
							model.LogMutexVan.Unlock()

							for i := currentLedger; i <= nextBatchEnd; i++ {
								if err := processStellarLedger(i, chainData.Config.HorizonURL); err != nil {
									model.LogMutexVan.Lock()
									log.Printf("Error processing ledger %d: %v, continuing...", i, err)
									model.LogMutexVan.Unlock()
								}
								time.Sleep(50 * time.Millisecond)
							}

							currentLedger = nextBatchEnd + 1
							model.ProcessLockVan.Lock()
							chainData.SetLastProcessedBlockVan(nextBatchEnd)
							model.ProcessLockVan.Unlock()

							model.LogMutexVan.Lock()
							log.Printf("✅ Fast scanned to ledger %d", nextBatchEnd)
							model.LogMutexVan.Unlock()
							continue
						}
					}
				}
			}
		}

		err := processStellarLedger(currentLedger, chainData.Config.HorizonURL)
		if err == nil {
			currentLedger++
			model.ProcessLockVan.Lock()
			if currentLedger-1 > chainData.GetLastProcessedBlockVan() {
				chainData.SetLastProcessedBlockVan(currentLedger - 1)
			}
			model.ProcessLockVan.Unlock()
			time.Sleep(200 * time.Millisecond)
		} else {
			sleepTime := 500 * time.Millisecond
			model.LogMutexVan.Lock()
			log.Printf("Waiting %v before retrying ledger %d", sleepTime, currentLedger)
			model.LogMutexVan.Unlock()
			time.Sleep(sleepTime)
		}
	}
}

// Quét từ hiện tại ngược về quá khứ// Quét từ hiện tại ngược về quá khứ
func reverseHandleStellarHTTP(chainName string, pastDuration time.Duration) {
	chainDataGeneric := configs.GetChainData(chainName)
	if chainDataGeneric == nil {
		log.Fatalf("Data not found for chain %s", chainName)
	}
	chainData := chainDataGeneric.GetConfigVan().(*model.ChainDataStellarVan)

	// Kiểm tra xem HorizonURL đã được cấu hình chưa
	if chainData.Config.HorizonURL == "" {
		log.Fatalf("HorizonURL is not configured for chain %s", chainName)
	}

	// Lấy thông tin ledger mới nhất từ Stellar Horizon API
	url := fmt.Sprintf("%s/ledgers?order=desc&limit=1", chainData.Config.HorizonURL)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error fetching chain status: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	var latestResp struct {
		Embedded struct {
			Records []model.StellarLedgerResponseVan `json:"records"`
		} `json:"_embedded"`
	}
	if err := json.Unmarshal(body, &latestResp); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	if len(latestResp.Embedded.Records) == 0 {
		log.Fatalf("No latest ledger found")
	}

	latestLedger := latestResp.Embedded.Records[0].Sequence
	latestLedgerTime := latestResp.Embedded.Records[0].Timestamp
	targetTime := latestLedgerTime.Add(-pastDuration)

	// Ghi log thông tin khởi đầu
	model.LogMutexVan.Lock()
	log.Printf("======= STARTING REVERSE SYSTEM FOR STELLAR CHAIN %s =======", chainName)
	log.Printf("Starting scan from current ledger = %d", latestLedger)
	log.Printf("Target time: %v", targetTime)
	log.Printf("==================================")
	model.LogMutexVan.Unlock()

	ledgerCounter := 0
	currentLedger := latestLedger

	// Bắt đầu quét ngược
	for {
		ledgerCounter++

		err := processStellarLedger(currentLedger, chainData.Config.HorizonURL)
		if err == nil {
			ledgerData, err := getStellarLedger(currentLedger, chainData.Config.HorizonURL)
			if err == nil {
				ledgerTime := ledgerData.Timestamp
				// Kiểm tra xem đã đạt đến thời gian mục tiêu chưa
				if ledgerTime.Before(targetTime) || ledgerTime.Equal(targetTime) {
					model.LogMutexVan.Lock()
					log.Printf("✅ Reached target time at ledger %d", currentLedger)
					model.LogMutexVan.Unlock()
					break
				}
			} else {
				model.LogMutexVan.Lock()
				log.Printf("⚠️ Error fetching ledger %d details: %v, continuing...", currentLedger, err)
				model.LogMutexVan.Unlock()
			}

			currentLedger--
			model.ProcessLockVan.Lock()
			chainData.SetLastProcessedBlockVan(currentLedger + 1)
			model.ProcessLockVan.Unlock()

			// Ghi log tiến trình sau mỗi 50 ledger
			if ledgerCounter%50 == 0 {
				model.LogMutexVan.Lock()
				log.Printf("🔄 Scanned %d ledgers backwards, currently at ledger %d", ledgerCounter, currentLedger)
				model.LogMutexVan.Unlock()
			}

			time.Sleep(200 * time.Millisecond)
		} else {
			// Xử lý lỗi khi không lấy được ledger, thử lại sau một khoảng thời gian
			sleepTime := 500 * time.Millisecond
			model.LogMutexVan.Lock()
			log.Printf("⚠️ Error processing ledger %d: %v", currentLedger, err)
			log.Printf("Waiting %v before retrying ledger %d", sleepTime, currentLedger)
			model.LogMutexVan.Unlock()
			time.Sleep(sleepTime)
		}

		// Kiểm tra xem đã đến ledger đầu tiên chưa
		if currentLedger <= 1 {
			model.LogMutexVan.Lock()
			log.Printf("✅ Reached the first ledger")
			model.LogMutexVan.Unlock()
			break
		}
	}

	// Ghi log hoàn tất
	model.LogMutexVan.Lock()
	log.Printf("✅ Completed processing ledgers backwards for chain %s", chainName)
	model.LogMutexVan.Unlock()
}

// Hàm chính xử lý Stellar qua HTTP
func Handle_stellar_http() {
	chainName := "stellar"
	logFile, err := os.OpenFile("./log/stellar_http.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Cannot open log file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	chainData := InitChainDataStellar(chainName)
	log.Printf("Initialized data for chain %s, last processed ledger: %d",
		chainName, chainData.GetLastProcessedBlockVan())

	if err := configs.LoadConfig("./services/get_chains/configs/config-stellar.json", chainName); err != nil {
		log.Fatalf("Cannot load config: %v", err)
	}

	if chainData.Config.HorizonURL == "" {
		log.Fatalf("HorizonURL is not set in config for chain %s", chainName)
	}

	pastDuration := 1 * time.Hour

	model.LogMutexVan.Lock()
	log.Printf("======= BEGIN PROCESSING STELLAR CHAIN %s =======", chainName)
	log.Printf("Step 1: Scanning backwards into the past (%v)", pastDuration)
	model.LogMutexVan.Unlock()

	reverseHandleStellarHTTP(chainName, pastDuration)

	model.LogMutexVan.Lock()
	log.Printf("Step 2: Continuing scan from past to present and monitoring new ledgers")
	model.LogMutexVan.Unlock()

	continueHandleStellarHTTP(chainName)
}
