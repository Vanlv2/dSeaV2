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

// Khởi tạo dữ liệu cho chuỗi Algorand
func InitAlgorandChainData(chainName string) *model.ChainDataAlgorand {
	if data, exists := model.ChainDataMapVan[chainName]; exists {
		return data.(*model.ChainDataAlgorand)
	}

	data := &model.ChainDataAlgorand{
		LastProcessedBlock: 0,
		LogData:            make(map[string]interface{}),
	}
	model.ChainDataMapVan[chainName] = data
	return data
}

// Lấy thông tin block từ Algorand blockchain qua API
func getAlgorandBlock(round int64, apiURL string) (*model.AlgorandBlockResponseVan, error) {
	url := fmt.Sprintf("%s/v2/blocks/%d", apiURL, round)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error calling Algorand API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var blockResp model.AlgorandBlockResponseVan
	if err := json.Unmarshal(body, &blockResp); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return &blockResp, nil
}

// Lấy giao dịch từ block bằng Indexer API
func getAlgorandTransactions(round int64, indexerURL string) ([]model.AlgorandTransactionVan, error) {
	url := fmt.Sprintf("%s/v2/transactions?min-round=%d&max-round=%d", indexerURL, round, round)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching transactions: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading transactions response: %v", err)
	}

	log.Printf("Raw transactions response for round %d: %s", round, string(body))

	var txResponse struct {
		Transactions []model.AlgorandTransactionVan `json:"transactions"`
	}
	if err := json.Unmarshal(body, &txResponse); err != nil {
		return nil, fmt.Errorf("error parsing transactions JSON: %v", err)
	}

	return txResponse.Transactions, nil
}

// Xử lý một block Algorand và xuất JSON theo định dạng mong muốn
func processAlgorandBlock(round int64, apiURL, indexerURL, chainName string) error {
	model.LogMutexVan.Lock()
	log.Printf("🔍 Fetching block info: %d", round)
	model.LogMutexVan.Unlock()

	block, err := getAlgorandBlock(round, apiURL)
	if err != nil {
		model.LogMutexVan.Lock()
		log.Printf("❌ Error fetching block %d: %v", round, err)
		model.LogMutexVan.Unlock()
		return err
	}

	model.LogMutexVan.Lock()
	log.Printf("✅ Fetched block: %d", round)
	model.LogMutexVan.Unlock()

	// Lấy giao dịch từ block bằng Indexer API
	transactions, err := getAlgorandTransactions(round, indexerURL)
	if err != nil {
		model.LogMutexVan.Lock()
		log.Printf("❌ Error fetching transactions for block %d: %v", round, err)
		model.LogMutexVan.Unlock()
		return err
	}

	// Tạo mảng JSON cho các giao dịch
	var txRecords []map[string]interface{}
	for _, tx := range transactions {
		txHash := tx.ID
		blockTime := time.Unix(block.Timestamp, 0).UTC()

		var receiver string
		var amount int64
		var token string = "ALGO" // Token mặc định

		// Xử lý dựa trên loại giao dịch
		switch tx.Type {
		case "pay": // Payment transaction
			if tx.PaymentTransaction != nil {
				receiver = tx.PaymentTransaction.Receiver
				amount = tx.PaymentTransaction.Amount
			}
		case "axfer": // Asset transfer transaction
			if tx.AssetTransferTransaction != nil {
				receiver = tx.AssetTransferTransaction.Receiver
				amount = tx.AssetTransferTransaction.Amount
				token = fmt.Sprintf("ASA-%d", tx.AssetTransferTransaction.AssetID) // Token là Asset ID
			}
		default:
			// Bỏ qua các loại giao dịch khác nếu không cần xử lý
			continue
		}

		// Nếu không có receiver hoặc amount hợp lệ, bỏ qua giao dịch này
		if receiver == "" {
			continue
		}

		if txHash == "" {
			txHash = fmt.Sprintf("%x", sha256.Sum256([]byte(tx.Sender+receiver+fmt.Sprintf("%d", amount))))
		}

		// Tạo bản ghi giao dịch theo định dạng mong muốn
		record := map[string]interface{}{
			"block_height": fmt.Sprintf("%d", block.Round),
			"block_hash":   block.Hash,
			"block_time":   blockTime.Format(time.RFC3339),
			"chain_id":     "algorand-mainnet", // Có thể thay đổi tùy theo cấu hình
			"tx_hash":      txHash,
			"from":         tx.Sender,
			"to":           receiver,
			"amount":       fmt.Sprintf("%d", amount),
			"token":        token,
			"total_amount": fmt.Sprintf("%d", amount), // Giả định total_amount = amount
			"tx_type":      tx.Type,
			"timestamp":    blockTime.Format(time.RFC3339),
		}
		txRecords = append(txRecords, record)
	}

	// Ghi JSON vào log theo từng giao dịch
	model.LogMutexVan.Lock()
	if len(txRecords) == 0 {
		log.Printf("Block #%d has no transactions", round)
	} else {
		for _, txRecord := range txRecords {
			txJSON, _ := json.MarshalIndent(txRecord, "", "  ")
			log.Printf("GIAO DỊCH TRONG BLOCK #%d:\n%s", round, string(txJSON))
		}
	}
	model.LogMutexVan.Unlock()

	model.LogMutexVan.Lock()
	log.Printf("✅ Completed processing block %d", round)
	model.LogMutexVan.Unlock()
	return nil
}

// Quét từ block hiện tại lùi về quá khứ
func scanBackwardsFromCurrentBlock(chainName string, blocksToScan int, apiURL string, indexerURL string) {
	// Lấy thông tin về block hiện tại
	url := fmt.Sprintf("%s/v2/status", apiURL)
	log.Printf("Fetching current block from: %s", url)
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
		LastRound int64 `json:"last-round"`
	}
	if err := json.Unmarshal(body, &status); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	currentBlock := status.LastRound
	log.Printf("Current block is: %d", currentBlock)

	// Xác định block bắt đầu và kết thúc
	startBlock := currentBlock
	endBlock := currentBlock - int64(blocksToScan)
	if endBlock < 1 {
		endBlock = 1 // Không quét dưới block 1
	}

	log.Printf("Starting backward scan from block %d to block %d", startBlock, endBlock)

	// Quét lùi từ block hiện tại
	for blockNum := startBlock; blockNum >= endBlock; blockNum-- {
		log.Printf("Processing block %d", blockNum)
		err := processAlgorandBlock(blockNum, apiURL, indexerURL, chainName)
		if err != nil {
			log.Printf("Error processing block %d: %v, retrying...", blockNum, err)
			// Thử lại sau 1 giây
			time.Sleep(1 * time.Second)
			err = processAlgorandBlock(blockNum, apiURL, indexerURL, chainName)
			if err != nil {
				log.Printf("Failed to process block %d after retry: %v, skipping", blockNum, err)
				continue
			}
		}

		// Tạm dừng giữa các lần xử lý block để tránh quá tải API
		time.Sleep(200 * time.Millisecond)
	}

	log.Printf("Completed backward scan from block %d to block %d", startBlock, endBlock)
}

// Hàm chính xử lý Algorand qua HTTP
func Handle_algorand_http() {
	chainName := "algorand"
	apiURL := "https://mainnet-api.algonode.cloud"
	indexerURL := "https://mainnet-idx.algonode.cloud"
	blocksToScan := 1000 // Số block cần quét lùi

	// Tạo thư mục log nếu chưa tồn tại
	if _, err := os.Stat("./log"); os.IsNotExist(err) {
		os.Mkdir("./log", 0755)
	}

	// Mở file log
	logFile, err := os.OpenFile("./services/get_chains/log/algorand_http.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Cannot open log file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	// Khởi tạo dữ liệu cho chain
	_ = InitAlgorandChainData(chainName)
	log.Printf("Initialized data for chain %s", chainName)

	// Tải cấu hình nếu cần
	if err := configs.LoadConfig("./configs/config-algorand.json", chainName); err != nil {
		log.Printf("Warning: Cannot load config: %v, using default settings", err)
	}

	// Ghi log bắt đầu quá trình
	log.Printf("======= BẮT ĐẦU QUÉT NGƯỢC BLOCKCHAIN ALGORAND %s =======", chainName)
	log.Printf("Sẽ quét lùi %d blocks từ block hiện tại", blocksToScan)
	log.Printf("API URL: %s", apiURL)
	log.Printf("Indexer URL: %s", indexerURL)
	log.Printf("==================================")

	// Bắt đầu quét lùi từ block hiện tại
	scanBackwardsFromCurrentBlock(chainName, blocksToScan, apiURL, indexerURL)

	// Ghi log kết thúc quá trình
	log.Printf("======= HOÀN THÀNH QUÉT NGƯỢC BLOCKCHAIN ALGORAND %s =======", chainName)
	log.Printf("Đã quét lùi %d blocks từ block hiện tại", blocksToScan)
	log.Printf("==================================")
}

// Hàm theo dõi các block mới (có thể thêm vào sau khi quét lùi hoàn tất)
func monitorNewBlocks(chainName string, apiURL string, indexerURL string) {
	log.Printf("Bắt đầu theo dõi các block mới trên blockchain Algorand")

	// Lấy thông tin về block hiện tại
	url := fmt.Sprintf("%s/v2/status", apiURL)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error fetching chain status: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}
	log.Printf("API response: %s", string(body))

	var status struct {
		LastRound int64 `json:"last-round"`
	}
	if err := json.Unmarshal(body, &status); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	currentBlock := status.LastRound
	log.Printf("Bắt đầu theo dõi từ block: %d", currentBlock)

	// Theo dõi liên tục các block mới
	for {
		// Kiểm tra block mới nhất
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Error fetching chain status: %v, retrying...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("Error reading response: %v, retrying...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var newStatus struct {
			LastRound int64 `json:"last-round"`
		}
		if err := json.Unmarshal(body, &newStatus); err != nil {
			log.Printf("Error parsing JSON: %v, retrying...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// Nếu có block mới
		if newStatus.LastRound > currentBlock {
			log.Printf("Phát hiện block mới: %d (trước đó: %d)", newStatus.LastRound, currentBlock)

			// Xử lý các block mới
			for blockNum := currentBlock + 1; blockNum <= newStatus.LastRound; blockNum++ {
				log.Printf("Xử lý block mới: %d", blockNum)
				err := processAlgorandBlock(blockNum, apiURL, indexerURL, chainName)
				if err != nil {
					log.Printf("Lỗi khi xử lý block %d: %v, thử lại...", blockNum, err)
					// Thử lại sau 1 giây
					time.Sleep(1 * time.Second)
					err = processAlgorandBlock(blockNum, apiURL, indexerURL, chainName)
					if err != nil {
						log.Printf("Không thể xử lý block %d sau khi thử lại: %v, bỏ qua", blockNum, err)
						continue
					}
				}

				// Cập nhật block hiện tại
				currentBlock = blockNum

				// Tạm dừng giữa các lần xử lý block để tránh quá tải API
				time.Sleep(200 * time.Millisecond)
			}
		}

		// Đợi một khoảng thời gian trước khi kiểm tra lại
		time.Sleep(5 * time.Second)
	}
}

// Hàm chính mở rộng để vừa quét lùi vừa theo dõi block mới
func Handle_algorand_http_extended() {
	chainName := "algorand"
	apiURL := "https://mainnet-api.algonode.cloud"
	indexerURL := "https://mainnet-idx.algonode.cloud"
	blocksToScan := 1000 // Số block cần quét lùi

	// Tạo thư mục log nếu chưa tồn tại
	if _, err := os.Stat("./log"); os.IsNotExist(err) {
		os.Mkdir("./log", 0755)
	}

	// Mở file log
	logFile, err := os.OpenFile("./log/algorand_http.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Cannot open log file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	// Khởi tạo dữ liệu cho chain
	_ = InitAlgorandChainData(chainName)
	log.Printf("Initialized data for chain %s", chainName)

	// Tải cấu hình nếu cần
	if err := configs.LoadConfig("./configs/config-algorand.json", chainName); err != nil {
		log.Printf("Warning: Cannot load config: %v, using default settings", err)
	}

	// Ghi log bắt đầu quá trình
	log.Printf("======= BẮT ĐẦU XỬ LÝ BLOCKCHAIN ALGORAND %s =======", chainName)
	log.Printf("Bước 1: Quét lùi %d blocks từ block hiện tại", blocksToScan)
	log.Printf("API URL: %s", apiURL)
	log.Printf("Indexer URL: %s", indexerURL)
	log.Printf("==================================")

	// Bước 1: Quét lùi từ block hiện tại
	scanBackwardsFromCurrentBlock(chainName, blocksToScan, apiURL, indexerURL)

	log.Printf("Bước 2: Bắt đầu theo dõi các block mới")

	// Bước 2: Theo dõi các block mới
	monitorNewBlocks(chainName, apiURL, indexerURL)
}
