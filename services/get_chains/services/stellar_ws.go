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

// Hàm chính xử lý Stellar qua HTTP, bắt đầu từ ledger mới nhất tại thời điểm hiện tại
func Handle_stellar_ws() {
	chainName := "stellar"
	logFile, err := os.OpenFile("./services/get_chains/log/stellar_ws.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Cannot open log file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	// Khởi tạo dữ liệu chain
	chainData := InitChainDataStellar(chainName)

	// Tải cấu hình
	if err := configs.LoadConfig("./services/get_chains/configs/config-stellar.json", chainName); err != nil {
		log.Fatalf("Cannot load config: %v", err)
	}

	if chainData.Config.HorizonURL == "" {
		log.Fatalf("HorizonURL is not set in config for chain %s", chainName)
	}

	// Lấy ledger mới nhất từ Stellar Horizon API để bắt đầu
	url := fmt.Sprintf("%s/ledgers?order=desc&limit=1", chainData.Config.HorizonURL)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("❌ Error fetching initial latest ledger: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("❌ Error reading response: %v", err)
	}

	var latestResp struct {
		Embedded struct {
			Records []model.StellarLedgerResponseVan `json:"records"`
		} `json:"_embedded"`
	}
	if err := json.Unmarshal(body, &latestResp); err != nil {
		log.Fatalf("❌ Error parsing JSON: %v", err)
	}

	if len(latestResp.Embedded.Records) == 0 {
		log.Fatalf("❌ No latest ledger found")
	}

	// Gán ledger mới nhất làm điểm bắt đầu
	latestLedger := latestResp.Embedded.Records[0].Sequence
	chainData.SetLastProcessedBlockVan(latestLedger)

	model.LogMutexVan.Lock()
	log.Printf("======= BEGIN MONITORING STELLAR CHAIN %s =======", chainName)
	log.Printf("Initialized data for chain %s, starting from latest ledger: %d", chainName, latestLedger)
	log.Printf("Monitoring new ledgers every 5 seconds with missed ledger recovery...")
	model.LogMutexVan.Unlock()

	// Vòng lặp quét ledger mới nhất và xử lý ledger bị bỏ sót
	for {
		// Lấy ledger mới nhất từ Stellar Horizon API
		resp, err := http.Get(url)
		if err != nil {
			model.LogMutexVan.Lock()
			log.Printf("❌ Error fetching latest ledger: %v. Retrying in 5 seconds...", err)
			model.LogMutexVan.Unlock()
			time.Sleep(5 * time.Second)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			model.LogMutexVan.Lock()
			log.Printf("❌ Error reading response: %v. Retrying in 5 seconds...", err)
			model.LogMutexVan.Unlock()
			time.Sleep(5 * time.Second)
			continue
		}

		if err := json.Unmarshal(body, &latestResp); err != nil {
			model.LogMutexVan.Lock()
			log.Printf("❌ Error parsing JSON: %v. Retrying in 5 seconds...", err)
			model.LogMutexVan.Unlock()
			time.Sleep(5 * time.Second)
			continue
		}

		if len(latestResp.Embedded.Records) == 0 {
			model.LogMutexVan.Lock()
			log.Printf("❌ No latest ledger found. Retrying in 5 seconds...")
			model.LogMutexVan.Unlock()
			time.Sleep(5 * time.Second)
			continue
		}

		latestLedger = latestResp.Embedded.Records[0].Sequence
		lastProcessedLedger := chainData.GetLastProcessedBlockVan()

		// Kiểm tra và xử lý các ledger bị bỏ sót hoặc ledger mới
		if latestLedger > lastProcessedLedger {
			missedLedgers := latestLedger - lastProcessedLedger
			if missedLedgers > 1 { // Có ledger bị bỏ sót
				model.LogMutexVan.Lock()
				log.Printf("⚠️ Detected %d missed ledgers. Processing from %d to %d...",
					missedLedgers-1, lastProcessedLedger+1, latestLedger)
				model.LogMutexVan.Unlock()
			} else {
				model.LogMutexVan.Lock()
				log.Printf("🔍 Detected new ledger. Processing ledger %d...", latestLedger)
				model.LogMutexVan.Unlock()
			}

			// Xử lý tất cả ledger từ lastProcessedLedger + 1 đến latestLedger
			for ledger := lastProcessedLedger + 1; ledger <= latestLedger; ledger++ {
				err := processStellarLedger(ledger, chainData.Config.HorizonURL)
				if err != nil {
					model.LogMutexVan.Lock()
					log.Printf("❌ Error processing ledger %d: %v. Skipping...", ledger, err)
					model.LogMutexVan.Unlock()
					continue
				}

				// Cập nhật ledger cuối cùng đã xử lý
				model.ProcessLockVan.Lock()
				chainData.SetLastProcessedBlockVan(ledger)
				model.ProcessLockVan.Unlock()
			}

			model.LogMutexVan.Lock()
			log.Printf("✅ Completed processing up to ledger %d", latestLedger)
			model.LogMutexVan.Unlock()
		} else {
			model.LogMutexVan.Lock()
			log.Printf("⏳ No new or missed ledgers detected. Latest ledger: %d, Last processed: %d", latestLedger, lastProcessedLedger)
			model.LogMutexVan.Unlock()
		}

		// Nghỉ 5 giây trước khi kiểm tra lại
		time.Sleep(5 * time.Second)
	}
}

// Các hàm hỗ trợ giữ nguyên
func InitChainDataStellarws(chainName string) *model.ChainDataStellarVan {
	if data, exists := model.ChainDataMapVan[chainName]; exists {
		return data.(*model.ChainDataStellarVan)
	}

	data := &model.ChainDataStellarVan{
		LastProcessedLedger: 0, // Giá trị mặc định ban đầu, sẽ được cập nhật sau
		LogData:             make(map[string]interface{}),
	}
	model.ChainDataMapVan[chainName] = data
	return data
}

func getStellarLedgerws(ledger int64, horizonURL string) (*model.StellarLedgerResponseVan, error) {
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

func getStellarTransactionsws(ledger int64, horizonURL string) ([]model.StellarTransactionVan, error) {
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

func processStellarLedgerws(ledger int64, horizonURL string) error {
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

	if txCount == 0 {
		event := model.TransactionEventVan{
			Address:         "",
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
		for _, tx := range transactions {
			txHash := fmt.Sprintf("%x", sha256.Sum256([]byte(tx.Source+tx.Destination+tx.Amount)))
			event := model.TransactionEventVan{
				Address:         tx.Destination,
				Amount:          tx.Amount,
				BlockNumber:     ledgerData.Sequence,
				EventSignature:  "",
				FromAddress:     tx.Source,
				LogIndex:        0,
				NameChain:       "stellar",
				RawData:         fmt.Sprintf("%v", tx),
				Timestamp:       tx.Timestamp.Format("2006-01-02 15:04:05"),
				ToAddress:       tx.Destination,
				TransactionType: "Transfer",
				TxHash:          txHash,
			}
			txEvents = append(txEvents, event)
		}
	}

	model.LogMutexVan.Lock()
	log.Printf("LEDGER MỚI #%d | Chain: %s | Time: %s | Hash: %s",
		ledgerData.Sequence, "stellar", ledgerData.Timestamp.Format("2006-01-02 15:04:05"), ledgerData.Hash)
	if txCount == 0 {
		log.Printf("Ledger #%d has no transactions", ledgerData.Sequence)
	} else {
		log.Printf("Number of transactions in ledger #%d: %d", ledgerData.Sequence, txCount)
	}
	for _, event := range txEvents {
		eventJSON, _ := json.MarshalIndent(event, "", "  ")
		log.Printf("EVENT TRONG LEDGER #%d:\n%s", ledgerData.Sequence, string(eventJSON))
	}
	log.Printf("✅ Completed processing ledger %d", ledger)
	model.LogMutexVan.Unlock()

	return nil
}
