package get_chains

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// Cấu trúc để lưu trữ lại khối xử lý cuối cùng
type CosmosChainData struct {
	LastProcessedBlock int64
	Config             ConfigCosmos
	LogData            map[string]interface{}
}

var cosmosChainDataMap = make(map[string]*CosmosChainData)
var cosmosProcessLock = &sync.Mutex{}

// Khởi tạo dữ liệu cho chuỗi Cosmos
func InitCosmosChainData(chainName string) *CosmosChainData {
	if data, exists := cosmosChainDataMap[chainName]; exists {
		return data
	}

	data := &CosmosChainData{
		LastProcessedBlock: 0,
		LogData:            make(map[string]interface{}),
	}
	cosmosChainDataMap[chainName] = data
	return data
}

// Lấy dữ liệu chuỗi Cosmos
func GetCosmosChainData(chainName string) *CosmosChainData {
	return cosmosChainDataMap[chainName]
}

// Tải cấu hình cho chuỗi Cosmos
func load_cosmos_config(filePath string, chainName string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("không thể đọc file cấu hình: %v", err)
	}

	chainData := GetCosmosChainData(chainName)
	if chainData == nil {
		return fmt.Errorf("không tìm thấy dữ liệu cho chain %s", chainName)
	}

	return json.Unmarshal(data, &chainData.Config)
}

// Cấu trúc cho phản hồi HTTP từ Cosmos RPC
type CosmosBlockResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		BlockID struct {
			Hash  string `json:"hash"`
			Parts struct {
				Total int    `json:"total"`
				Hash  string `json:"hash"`
			} `json:"parts"`
		} `json:"block_id"`
		Block struct {
			Header struct {
				Version struct {
					Block string `json:"block"`
				} `json:"version"`
				ChainID     string    `json:"chain_id"`
				Height      string    `json:"height"`
				Time        time.Time `json:"time"`
				LastBlockID struct {
					Hash  string `json:"hash"`
					Parts struct {
						Total int    `json:"total"`
						Hash  string `json:"hash"`
					} `json:"parts"`
				} `json:"last_block_id"`
				LastCommitHash     string `json:"last_commit_hash"`
				DataHash           string `json:"data_hash"`
				ValidatorsHash     string `json:"validators_hash"`
				NextValidatorsHash string `json:"next_validators_hash"`
				ConsensusHash      string `json:"consensus_hash"`
				AppHash            string `json:"app_hash"`
				LastResultsHash    string `json:"last_results_hash"`
				EvidenceHash       string `json:"evidence_hash"`
				ProposerAddress    string `json:"proposer_address"`
			} `json:"header"`
			Data struct {
				Txs []string `json:"txs"` // Giao dịch dạng base64
			} `json:"data"`
			Evidence struct {
				Evidence []interface{} `json:"evidence"`
			} `json:"evidence"`
			LastCommit struct {
				Height  string `json:"height"`
				Round   int    `json:"round"`
				BlockID struct {
					Hash  string `json:"hash"`
					Parts struct {
						Total int    `json:"total"`
						Hash  string `json:"hash"`
					} `json:"parts"`
				} `json:"block_id"`
				Signatures []struct {
					BlockIDFlag      int       `json:"block_id_flag"`
					ValidatorAddress string    `json:"validator_address"`
					Timestamp        time.Time `json:"timestamp"`
					Signature        string    `json:"signature"`
				} `json:"signatures"`
			} `json:"last_commit"`
		} `json:"block"`
	} `json:"result"`
}

// Cấu trúc cho phản hồi TxSearch từ Cosmos RPC
type CosmosTxSearchResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		Txs []struct {
			Hash     string `json:"hash"`
			Height   string `json:"height"`
			Index    int    `json:"index"`
			TxResult struct {
				Code      int    `json:"code"`
				Data      string `json:"data"`
				Log       string `json:"log"`
				Info      string `json:"info"`
				GasWanted string `json:"gas_wanted"`
				GasUsed   string `json:"gas_used"`
				Events    []struct {
					Type       string `json:"type"`
					Attributes []struct {
						Key   string `json:"key"`
						Value string `json:"value"`
						Index bool   `json:"index"`
					} `json:"attributes"`
				} `json:"events"`
				Codespace string `json:"codespace"`
			} `json:"tx_result"`
			Tx string `json:"tx"`
		} `json:"txs"`
		TotalCount string `json:"total_count"`
	} `json:"result"`
}

// Lấy thông tin block từ Cosmos blockchain
func getCosmosBlock(height int64, rpcURL string) (*CosmosBlockResponse, error) {
	url := fmt.Sprintf("%s/block?height=%d", rpcURL, height)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi gọi API Cosmos: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi đọc phản hồi: %v", err)
	}

	var blockResp CosmosBlockResponse
	if err := json.Unmarshal(body, &blockResp); err != nil {
		return nil, fmt.Errorf("lỗi khi parse JSON: %v", err)
	}

	return &blockResp, nil
}

// Lấy thông tin giao dịch trong một block
func getCosmosTxsForBlock(height int64, rpcURL string) (*CosmosTxSearchResponse, error) {
	url := fmt.Sprintf("%s/tx_search?query=\"tx.height=%d\"&prove=true", rpcURL, height)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi gọi API Cosmos để lấy giao dịch: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi đọc phản hồi: %v", err)
	}

	var txResp CosmosTxSearchResponse
	if err := json.Unmarshal(body, &txResp); err != nil {
		return nil, fmt.Errorf("lỗi khi parse JSON: %v", err)
	}

	return &txResp, nil
}

// Ghi block vào file log - giữ nguyên
func writeCosmosBlockToFile(height int64, block *CosmosBlockResponse, txCount int, chainName string) {
	filePath := fmt.Sprintf("./services/get_chains/log/block_data_cosmos_%s.log", chainName)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Không thể mở file %s: %v", filePath, err)
		return
	}
	defer file.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	blockData := fmt.Sprintf("[%s] Block: %d, Transactions: %d\n", timestamp, height, txCount)

	blockJSON, err := json.MarshalIndent(block, "", "  ")
	if err != nil {
		log.Printf("Không thể chuyển đổi block thành JSON: %v", err)
	} else {
		blockData += string(blockJSON) + "\n----------------------------------------\n"
	}

	if _, err := file.WriteString(blockData); err != nil {
		log.Printf("Không thể ghi vào file %s: %v", filePath, err)
	} else {
		log.Printf("Đã ghi khối %d (%d giao dịch) vào file cho chain %s", height, txCount, chainName)
	}
}

// Trích xuất dữ liệu giao dịch từ Cosmos - Sử dụng phương pháp chuyển đổi JSON
func extractCosmosTransactionData(tx interface{}, chainName string, blockTime time.Time, height int64, blockHash string) TransactionRecord {
	// Phân tích dữ liệu giao dịch theo định dạng của Cosmos
	txData, ok := tx.(map[string]interface{})
	if !ok {
		return TransactionRecord{}
	}

	// Trích xuất thông tin từ giao dịch Cosmos
	txHash, _ := txData["hash"].(string)

	// Trích xuất các thông tin từ events
	var from, to, amount, token, totalAmount string

	// Kiểm tra xem tx_result có tồn tại không
	if txResult, exists := txData["tx_result"]; exists {
		// Chuyển đổi struct thành JSON và sau đó parse lại
		txResultBytes, err := json.Marshal(txResult)
		if err == nil {
			var txResultMap map[string]interface{}
			if json.Unmarshal(txResultBytes, &txResultMap) == nil {
				if events, hasEvents := txResultMap["events"].([]interface{}); hasEvents {
					for _, event := range events {
						eventMap, ok := event.(map[string]interface{})
						if !ok {
							continue
						}

						eventType, _ := eventMap["type"].(string)
						if eventType == "transfer" {
							attributes, hasAttr := eventMap["attributes"].([]interface{})
							if !hasAttr {
								continue
							}

							for _, attr := range attributes {
								attrMap, ok := attr.(map[string]interface{})
								if !ok {
									continue
								}

								key, _ := attrMap["key"].(string)
								value, _ := attrMap["value"].(string)

								if key == "sender" {
									from = value
								} else if key == "recipient" {
									to = value
								} else if key == "amount" {
									totalAmount = value
									amount, token = parseAmount(value)
								}
							}
						}
					}
				}
			}
		}
	}

	// Tạo và trả về đối tượng TransactionRecord giống như trong handle_cosmos_ws.go
	return TransactionRecord{
		BlockHeight: fmt.Sprintf("%d", height),
		BlockHash:   blockHash,
		BlockTime:   blockTime,
		ChainID:     chainName,
		TxHash:      txHash,
		From:        from,
		To:          to,
		Amount:      amount,
		Token:       token,
		TotalAmount: totalAmount,
		TxType:      "Transfer",
		Timestamp:   blockTime.Format(time.RFC3339), // Thêm timestamp
	}
}

// Xử lý một block Cosmos - Thay đổi để có cấu trúc log giống ws
func processCosmosBlock(height int64, chainName string, rpcURL string) error {
	logMutex.Lock()
	log.Printf("🔍 Đang lấy thông tin khối: %d", height)
	logMutex.Unlock()

	// Lấy thông tin block
	block, err := getCosmosBlock(height, rpcURL)
	if err != nil {
		logMutex.Lock()
		log.Printf("❌ Lỗi khi lấy khối %d: %v", height, err)
		logMutex.Unlock()
		return err
	}

	logMutex.Lock()
	log.Printf("✅ Đã lấy khối: %d", height)
	logMutex.Unlock()

	// Lấy giao dịch trong block
	txsResp, err := getCosmosTxsForBlock(height, rpcURL)
	if err != nil {
		logMutex.Lock()
		log.Printf("❌ Lỗi khi lấy giao dịch cho khối %d: %v", height, err)
		logMutex.Unlock()
		return err
	}

	txCount := len(txsResp.Result.Txs)
	blockHeight, _ := strconv.ParseInt(block.Result.Block.Header.Height, 10, 64)
	blockTime := block.Result.Block.Header.Time
	blockHash := block.Result.BlockID.Hash

	// In thông tin block như trong handle_cosmos_ws.go
	logMutex.Lock()
	log.Printf("BLOCK MỚI #%s | Chain: %s | Time: %s | Hash: %s",
		block.Result.Block.Header.Height, block.Result.Block.Header.ChainID,
		blockTime.Format(time.RFC3339), blockHash)
	logMutex.Unlock()

	if txCount == 0 {
		logMutex.Lock()
		log.Printf("Block #%s không có giao dịch", block.Result.Block.Header.Height)
		logMutex.Unlock()
		return nil
	}

	logMutex.Lock()
	log.Printf("Số lượng giao dịch trong block #%s: %d", block.Result.Block.Header.Height, txCount)
	logMutex.Unlock()

	// Ghi block vào file log
	writeCosmosBlockToFile(height, block, txCount, chainName)

	// Tạo danh sách các giao dịch - tương tự như trong handle_cosmos_ws.go
	var transactions []TransactionRecord

	// Xử lý từng giao dịch
	for _, tx := range txsResp.Result.Txs {
		txMap := map[string]interface{}{
			"hash":      tx.Hash,
			"height":    tx.Height,
			"tx_result": tx.TxResult,
			"tx":        tx.Tx,
		}

		// Trích xuất dữ liệu từ transaction dưới dạng TransactionRecord
		record := extractCosmosTransactionData(txMap, chainName, blockTime, blockHeight, blockHash)

		// Thêm vào danh sách giao dịch
		if record.TxHash != "" {
			transactions = append(transactions, record)
		}
	}

	// Ghi log các giao dịch theo định dạng JSON giống như trong handle_cosmos_ws.go
	for _, record := range transactions {
		recordJSON, _ := json.MarshalIndent(record, "", "  ")
		logMutex.Lock()
		log.Printf("GIAO DỊCH TRONG BLOCK #%s:\n%s", block.Result.Block.Header.Height, string(recordJSON))
		logMutex.Unlock()
	}

	logMutex.Lock()
	log.Printf("✅ Hoàn thành xử lý khối %d", height)
	logMutex.Unlock()
	return nil
}

// Xử lý các block từ quá khứ đến hiện tại
func continueHandleCosmosHTTP(chainName string) {
	chainData := GetCosmosChainData(chainName)
	if chainData == nil {
		log.Fatalf("Không tìm thấy dữ liệu cho chain %s", chainName)
	}

	// Khởi tạo block bắt đầu (có thể cấu hình trong file)
	startBlockNumber := int64(1)
	chainData.LastProcessedBlock = startBlockNumber - 1

	logMutex.Lock()
	log.Printf("======= KHỞI ĐỘNG HỆ THỐNG CHO CHAIN COSMOS %s =======", chainName)
	log.Printf("Khởi tạo lastProcessedBlock = %d", chainData.LastProcessedBlock)
	log.Printf("Bắt đầu quét từ khối = %d", startBlockNumber)
	log.Printf("==================================")
	logMutex.Unlock()

	blockCounter := 0
	currentBlock := startBlockNumber

	for {
		blockCounter++

		// Mỗi 50 khối, kiểm tra xem có bị bỏ lỡ khối nào không
		if blockCounter%50 == 0 {
			// Lấy block hiện tại từ API
			url := fmt.Sprintf("%s/status", chainData.Config.RPC)
			resp, err := http.Get(url)

			if err == nil {
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)

				if err == nil {
					var statusResp struct {
						JSONRPC string `json:"jsonrpc"`
						ID      int    `json:"id"`
						Result  struct {
							SyncInfo struct {
								LatestBlockHeight string `json:"latest_block_height"`
							} `json:"sync_info"`
						} `json:"result"`
					}

					if json.Unmarshal(body, &statusResp) == nil {
						latestBlock, err := strconv.ParseInt(statusResp.Result.SyncInfo.LatestBlockHeight, 10, 64)
						if err == nil {
							gap := latestBlock - currentBlock
							if gap > 20 {
								// Có khoảng cách lớn, xử lý nhanh các khối bị bỏ lỡ
								nextBatchEnd := currentBlock + 100
								if nextBatchEnd > latestBlock {
									nextBatchEnd = latestBlock
								}

								logMutex.Lock()
								log.Printf("⚠️ Phát hiện khoảng cách %d khối trên chain %s. Đang quét nhanh từ %d đến %d...",
									gap, chainName, currentBlock, nextBatchEnd)
								logMutex.Unlock()

								for i := currentBlock; i <= nextBatchEnd; i++ {
									if err := processCosmosBlock(i, chainName, chainData.Config.RPC); err != nil {
										logMutex.Lock()
										log.Printf("Lỗi khi xử lý khối %d: %v, tiếp tục...", i, err)
										logMutex.Unlock()
									}
									time.Sleep(50 * time.Millisecond)
								}

								currentBlock = nextBatchEnd + 1

								cosmosProcessLock.Lock()
								chainData.LastProcessedBlock = nextBatchEnd
								cosmosProcessLock.Unlock()

								logMutex.Lock()
								log.Printf("✅ Đã quét nhanh đến khối %d", nextBatchEnd)
								logMutex.Unlock()
								continue
							}
						}
					}
				}
			}
		}

		// Xử lý khối tiếp theo
		err := processCosmosBlock(currentBlock, chainName, chainData.Config.RPC)
		if err == nil {
			currentBlock++

			cosmosProcessLock.Lock()
			if currentBlock-1 > chainData.LastProcessedBlock {
				chainData.LastProcessedBlock = currentBlock - 1
			}
			cosmosProcessLock.Unlock()

			time.Sleep(200 * time.Millisecond)
		} else {
			sleepTime := 500 * time.Millisecond
			logMutex.Lock()
			log.Printf("Đợi %v trước khi thử lại khối %d", sleepTime, currentBlock)
			logMutex.Unlock()
			time.Sleep(sleepTime)
		}
	}
}

// Xử lý các block từ hiện tại ngược về quá khứ
func reverseHandleCosmosHTTP(chainName string, pastDuration time.Duration) {
	chainData := GetCosmosChainData(chainName)
	if chainData == nil {
		log.Fatalf("Không tìm thấy dữ liệu cho chain %s", chainName)
	}

	// Lấy block hiện tại
	url := fmt.Sprintf("%s/status", chainData.Config.RPC)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Lỗi khi lấy trạng thái chain: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Lỗi khi đọc phản hồi: %v", err)
	}

	var statusResp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			SyncInfo struct {
				LatestBlockHeight string `json:"latest_block_height"`
				LatestBlockTime   string `json:"latest_block_time"`
			} `json:"sync_info"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &statusResp); err != nil {
		log.Fatalf("Lỗi khi parse JSON: %v", err)
	}

	latestBlock, err := strconv.ParseInt(statusResp.Result.SyncInfo.LatestBlockHeight, 10, 64)
	if err != nil {
		log.Fatalf("Lỗi khi chuyển đổi số khối: %v", err)
	}

	latestBlockTime, err := time.Parse(time.RFC3339, statusResp.Result.SyncInfo.LatestBlockTime)
	if err != nil {
		latestBlockTime = time.Now()
	}

	// Tính thời gian mục tiêu
	targetTime := latestBlockTime.Add(-pastDuration)

	logMutex.Lock()
	log.Printf("======= KHỞI ĐỘNG HỆ THỐNG NGƯỢC CHO CHAIN COSMOS %s =======", chainName)
	log.Printf("Bắt đầu quét từ khối hiện tại = %d", latestBlock)
	log.Printf("Thời gian mục tiêu: %v", targetTime)
	log.Printf("==================================")
	logMutex.Unlock()

	blockCounter := 0
	currentBlock := latestBlock

	for {
		blockCounter++

		// Xử lý khối hiện tại
		err := processCosmosBlock(currentBlock, chainName, chainData.Config.RPC)
		if err == nil {
			// Lấy thông tin block để kiểm tra điều kiện dừng
			block, err := getCosmosBlock(currentBlock, chainData.Config.RPC)
			if err == nil {
				blockTime := block.Result.Block.Header.Time

				// Kiểm tra nếu đã đạt đến thời gian mục tiêu
				if blockTime.Before(targetTime) || blockTime.Equal(targetTime) {
					logMutex.Lock()
					log.Printf("✅ Đã đạt đến thời gian mục tiêu tại khối %d", currentBlock)
					logMutex.Unlock()
					break
				}
			}

			// Giảm số block để quét ngược về quá khứ
			currentBlock--

			// Cập nhật LastProcessedBlock
			cosmosProcessLock.Lock()
			chainData.LastProcessedBlock = currentBlock + 1
			cosmosProcessLock.Unlock()

			// Mỗi 50 khối, in thông tin tiến độ
			if blockCounter%50 == 0 {
				logMutex.Lock()
				log.Printf("🔄 Đã quét ngược %d khối, hiện tại ở khối %d", blockCounter, currentBlock)
				logMutex.Unlock()
			}

			time.Sleep(200 * time.Millisecond)
		} else {
			sleepTime := 500 * time.Millisecond
			logMutex.Lock()
			log.Printf("Đợi %v trước khi thử lại khối %d", sleepTime, currentBlock)
			logMutex.Unlock()
			time.Sleep(sleepTime)
		}

		// Dừng khi đã quét đến khối 1
		if currentBlock <= 1 {
			logMutex.Lock()
			log.Printf("✅ Đã quét đến khối đầu tiên")
			logMutex.Unlock()
			break
		}
	}

	logMutex.Lock()
	log.Printf("✅ Hoàn thành xử lý các block ngược cho chain %s", chainName)
	logMutex.Unlock()
}

func handle_cosmos_http() {
	chainName := "cosmos"

	// Mở file log giống như trong handle_cosmos_ws.go
	logFile, err := os.OpenFile("./services/get_chains/log/cosmos_http.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Không thể mở file log: %v", err)
	}
	defer logFile.Close()

	// Chuyển hướng log vào file
	log.SetOutput(logFile)

	// Khởi tạo chainData và sử dụng ngay sau khi khai báo
	chainData := InitCosmosChainData(chainName)
	log.Printf("Khởi tạo dữ liệu cho chain %s, khối cuối cùng: %d",
		chainName, chainData.LastProcessedBlock)

	// Tải cấu hình
	if err := load_cosmos_config("./services/get_chains/config_chain/config-cosmos.json", chainName); err != nil {
		log.Fatalf("Không thể tải cấu hình: %v", err)
	}

	// Xử lý các khối từ giờ trở về 1 giờ trước
	pastDuration := 1 * time.Hour

	logMutex.Lock()
	log.Printf("======= BẮT ĐẦU QUÁ TRÌNH XỬ LÝ CHAIN COSMOS %s =======", chainName)
	log.Printf("Bước 1: Quét ngược về quá khứ (%v)", pastDuration)
	logMutex.Unlock()

	// Bắt đầu quét ngược từ hiện tại về quá khứ
	reverseHandleCosmosHTTP(chainName, pastDuration)

	logMutex.Lock()
	log.Printf("Bước 2: Tiếp tục quét từ quá khứ đến hiện tại và theo dõi các khối mới")
	logMutex.Unlock()

	// Sau đó tiếp tục quét từ quá khứ đến hiện tại
	continueHandleCosmosHTTP(chainName)
}
