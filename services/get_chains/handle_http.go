package get_chains

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func extractTransactionData(tx map[string]interface{}, chainName string, blockTime string) map[string]interface{} {
	logData := make(map[string]interface{})

	// Lấy dữ liệu cơ bản từ transaction
	txHash, _ := tx["hash"].(string)
	from, _ := tx["from"].(string)
	to, _ := tx["to"].(string)
	blockNumberHex, _ := tx["blockNumber"].(string)
	input, _ := tx["input"].(string)
	valueHex, _ := tx["value"].(string)

	// Chuyển đổi giá trị và block number
	value := new(big.Int)
	if len(valueHex) > 2 {
		value.SetString(valueHex[2:], 16)
	}

	blockNumber := new(big.Int)
	if len(blockNumberHex) > 2 {
		blockNumber.SetString(blockNumberHex[2:], 16)
	}

	// Cấu trúc dữ liệu để lưu vào database
	logData["name_chain"] = chainName
	logData["timestamp"] = blockTime
	logData["block_number"] = blockNumber.Uint64()
	logData["tx_hash"] = txHash
	logData["address"] = to
	logData["from_address"] = from
	logData["to_address"] = to
	logData["amount"] = value.String()
	logData["raw_data"] = fmt.Sprintf("%v", tx)

	// Xử lý input data và signature
	if len(input) >= 10 {
		logData["event_signature"] = input[:10]

		// Phân tích event signature để lấy transaction_type
		if transactionType, err := Parse_event_signature_name(input[:10]); err == nil {
			logData["transaction_type"] = transactionType
		} else {
			log.Printf("Không thể parse event signature %s: %v", input[:10], err)
			logData["transaction_type"] = "Unknown"
		}
	} else {
		logData["event_signature"] = ""
		logData["transaction_type"] = "Transfer"
	}

	return logData
}

func importLogFileToDatabase(logFilePath string, chainName string) error {
	log.Printf("🔄 Bắt đầu nhập dữ liệu từ file %s vào database...", logFilePath)

	data, err := os.ReadFile(logFilePath)
	if err != nil {
		return fmt.Errorf("không thể đọc file log: %v", err)
	}

	// Tách file thành các block riêng biệt
	blocks := strings.Split(string(data), "----------------------------------------")

	for _, blockData := range blocks {
		if strings.TrimSpace(blockData) == "" {
			continue
		}

		lines := strings.Split(blockData, "\n")
		if len(lines) < 2 {
			continue
		}

		// Parse dòng header để lấy thông tin block
		headerLine := strings.TrimSpace(lines[0])
		headerParts := strings.Split(headerLine, "] ")
		if len(headerParts) < 2 {
			continue
		}

		timestamp := strings.Trim(headerParts[0], "[")
		blockInfo := headerParts[1]

		blockNumberStr := ""
		if parts := strings.Split(blockInfo, ", "); len(parts) > 0 {
			blockNumberStr = strings.TrimPrefix(parts[0], "Block: ")
		}

		// Parse JSON block data
		var block map[string]interface{}
		jsonStart := strings.Index(blockData, "{")
		if jsonStart == -1 {
			continue
		}

		jsonData := blockData[jsonStart:]
		if err := json.Unmarshal([]byte(jsonData), &block); err != nil {
			log.Printf("Lỗi khi parse JSON của block: %v", err)
			continue
		}

		// Lấy transactions từ block
		transactions, ok := block["transactions"].([]interface{})
		if !ok {
			continue
		}

		for _, tx := range transactions {
			txMap, ok := tx.(map[string]interface{})
			if !ok {
				continue
			}

			// Trích xuất dữ liệu transaction
			logData := extractTransactionData(txMap, chainName, timestamp)
			blockNumber, _ := new(big.Int).SetString(blockNumberStr, 10)
			if blockNumber != nil {
				logData["block_number"] = blockNumber.Uint64()
			}
		}
	}

	log.Printf("✅ Hoàn thành nhập dữ liệu từ file %s vào database", logFilePath)
	return nil
}

// Hàm ghi block vào file
func write_block_to_file(blockNumber *big.Int, block map[string]interface{}, txCount int, chainName string) {
	filePath := fmt.Sprintf("./services/get_chains/block_data_%s.log", chainName)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Không thể mở file %s: %v", filePath, err)
		return
	}
	defer file.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	blockData := fmt.Sprintf("[%s] Block: %s, Transactions: %d\n", timestamp, blockNumber.String(), txCount)

	blockJSON, err := json.MarshalIndent(block, "", "  ")
	if err != nil {
		log.Printf("Không thể chuyển đổi block thành JSON: %v", err)
	} else {
		blockData += string(blockJSON) + "\n----------------------------------------\n"
	}

	if _, err := file.WriteString(blockData); err != nil {
		log.Printf("Không thể ghi vào file %s: %v", filePath, err)
	} else {
		log.Printf("Đã ghi khối %s (%d giao dịch) vào file cho chain %s", blockNumber.String(), txCount, chainName)
	}
}

// Hàm ghi transaction vào file
func writeTransactionToFile(tx map[string]interface{}) {
	txHash, _ := tx["hash"].(string)
	if txHash == "" {
		log.Printf("Bỏ qua ghi log cho giao dịch không có hash")
		return
	}

	filePath := "./transactions.log"
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Không thể mở file %s: %v", filePath, err)
		return
	}
	defer file.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	txData := fmt.Sprintf("[%s] Transaction: %s\n", timestamp, txHash)

	txJSON, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		log.Printf("Không thể chuyển đổi transaction thành JSON: %v", err)
	} else {
		txData += string(txJSON) + "\n----------------------------------------\n"
	}

	if _, err := file.WriteString(txData); err != nil {
		log.Printf("Không thể ghi vào file %s: %v", filePath, err)
	} else {
		log.Printf("Đã ghi giao dịch %s vào file", txHash)
	}
}

// Xử lý block
func processBlock(client *rpc.Client, blockNumber *big.Int, chainName string) error {
	var block map[string]interface{}
	blockHex := fmt.Sprintf("0x%x", blockNumber)

	log.Printf("🔍 Đang lấy thông tin khối: %s", blockHex)

	err := client.Call(&block, "eth_getBlockByNumber", blockHex, true)
	if err != nil {
		log.Printf("❌ Lỗi khi lấy khối %s: %v", blockHex, err)
		return err
	}

	log.Printf("✅ Đã lấy khối: %s - %s", blockHex, blockNumber.String())

	if block["transactions"] == nil {
		log.Printf("⚠️ Khối %s không có giao dịch nào", blockHex)
		return fmt.Errorf("không tìm thấy giao dịch")
	}

	transactions := block["transactions"].([]interface{})
	txCount := len(transactions)

	// Lấy thời gian của block
	blockTimeHex, _ := block["timestamp"].(string)
	blockTimeInt := new(big.Int)
	if len(blockTimeHex) > 2 {
		blockTimeInt.SetString(blockTimeHex[2:], 16)
	}

	write_block_to_file(blockNumber, block, txCount, chainName)

	log.Printf("🔄 Đang xử lý %d giao dịch từ khối %s", txCount, blockNumber.String())

	for i, tx := range transactions {
		txMap := tx.(map[string]interface{})
		txHash, _ := txMap["hash"].(string)
		log.Printf("   📝 Xử lý giao dịch (%d/%d): %s", i+1, txCount, txHash)

		// Vẫn giữ lại việc ghi log ra file nếu cần
		writeTransactionToFile(txMap)
	}

	log.Printf("✅ Hoàn thành xử lý khối %s - %s", blockHex, blockNumber.String())
	return nil
}

func importAllLogFiles() {
	chains := []string{"bsc", "avalanche"}

	for _, chain := range chains {
		logFilePath := fmt.Sprintf("./block_data_%s.log", chain)
		if _, err := os.Stat(logFilePath); err == nil {
			log.Printf("Tìm thấy file log cho chain %s, bắt đầu nhập dữ liệu...", chain)
			if err := importLogFileToDatabase(logFilePath, chain); err != nil {
				log.Printf("❌ Lỗi khi nhập dữ liệu từ %s: %v", logFilePath, err)
			}
		}
	}
}

// Xử lý transaction
func processTransaction(tx map[string]interface{}, chainName string) {
	to, ok := tx["to"].(string)
	if !ok || to == "" {
		log.Printf("   ⏩ Bỏ qua giao dịch tạo hợp đồng")
		return
	}

	txHash, ok := tx["hash"].(string)
	if !ok || txHash == "" {
		log.Printf("   ⏩ Bỏ qua giao dịch không hợp lệ")
		return
	}

	toAddress := strings.ToLower(to)
	valueHex, _ := tx["value"].(string)
	value := new(big.Int)
	if len(valueHex) > 2 {
		value.SetString(valueHex[2:], 16)
	}

	input, _ := tx["input"].(string)
	from, _ := tx["from"].(string)

	log.Printf("   💼 Giao dịch: %s", txHash)
	log.Printf("   📤 Từ: %s", from)
	log.Printf("   📥 Đến: %s", toAddress)
	log.Printf("   💰 Giá trị: %s wei", value.String())

	chainData := GetChainData(chainName)
	if chainData == nil {
		log.Printf("Không tìm thấy dữ liệu cho chain %s", chainName)
		return
	}

	if len(input) >= 10 && input[:10] == chainData.Config.TransferSignature {
		log.Printf("💰 GIAO DỊCH TRANSFER PHÁT HIỆN 💰")
		log.Printf("   Chain: %s", chainName)
		log.Printf("   Contract: %s", toAddress)
		log.Printf("   TxHash: %s", txHash)
		log.Printf("   Input: %s", input[:20]+"...")

		process_transfer_and_save(chainName, tx)
	}
}

// Xử lý và lưu thông tin transfer
func process_transfer_and_save(chainName string, tx map[string]interface{}) {
	chainData := GetChainData(chainName)
	if chainData == nil {
		log.Printf("Không tìm thấy dữ liệu cho chain %s", chainName)
		return
	}

	blockTimeHex, _ := tx["timestamp"].(string)
	if blockTimeHex == "" {
		blockTimeHex = "0x" + strconv.FormatInt(time.Now().Unix(), 16)
	}

	blockTimeInt := new(big.Int)
	if len(blockTimeHex) > 2 {
		blockTimeInt.SetString(blockTimeHex[2:], 16)
	}
	blockTime := time.Unix(blockTimeInt.Int64(), 0).Format("2006-01-02 15:04:05")

	// Trích xuất dữ liệu
	logData := extractTransactionData(tx, chainName, blockTime)

	// Cập nhật LogData của chain
	chainData.LogData = logData
}

// Hàm duyệt ngược từ thời gian hiện tại về quá khứ
func processBlocksInReverse(client *rpc.Client, pastDuration time.Duration) (map[string]interface{}, error) {
	// Tính toán thời gian mục tiêu
	targetTime := time.Now().Add(-pastDuration)

	// Lấy block hiện tại
	var latestBlockHex string
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := client.CallContext(ctx, &latestBlockHex, "eth_blockNumber")
	cancel()
	if err != nil {
		return nil, fmt.Errorf("lỗi khi lấy block hiện tại: %v", err)
	}

	latestBlock := new(big.Int)
	latestBlock.SetString(latestBlockHex[2:], 16)

	// Khởi tạo map để lưu dữ liệu
	blockDataMap := make(map[string]interface{})

	// Duyệt ngược từ block hiện tại về quá khứ
	for {
		var block map[string]interface{}
		blockHex := fmt.Sprintf("0x%x", latestBlock)

		err := client.Call(&block, "eth_getBlockByNumber", blockHex, true)
		if err != nil {
			log.Printf("❌ Lỗi khi lấy khối %s: %v", blockHex, err)
			break
		}

		// Lấy thời gian của block
		blockTimeHex, _ := block["timestamp"].(string)
		blockTimeInt := new(big.Int)
		if len(blockTimeHex) > 2 {
			blockTimeInt.SetString(blockTimeHex[2:], 16)
		}
		blockTime := time.Unix(blockTimeInt.Int64(), 0)
		fmt.Printf("Thời gian của block: %s", blockTime)

		// Kiểm tra nếu blockTime nhỏ hơn hoặc bằng targetTime thì dừng
		if blockTime.Before(targetTime) || blockTime.Equal(targetTime) {
			break
		}

		// Thu thập dữ liệu của block vào map
		blockDataMap[blockHex] = block

		// Giảm số block để duyệt ngược
		latestBlock.Sub(latestBlock, big.NewInt(1))
	}

	return blockDataMap, nil
}

// Khởi động xử lý HTTP
func continueHandleHTTP(client *ethclient.Client, chainName string) {
	chainData := GetChainData(chainName)
	if chainData == nil {
		log.Fatalf("Không tìm thấy dữ liệu cho chain %s", chainName)
	}

	// Khởi tạo block bắt đầu (có thể cấu hình trong file)
	blockNumber := big.NewInt(21604935)
	chainData.LastProcessedBlock = new(big.Int).Set(blockNumber)
	chainData.LastProcessedBlock.Sub(chainData.LastProcessedBlock, big.NewInt(1))

	log.Printf("======= KHỞI ĐỘNG HỆ THỐNG CHO CHAIN %s =======", chainName)
	log.Printf("Khởi tạo lastProcessedBlock = %d", chainData.LastProcessedBlock)
	log.Printf("Bắt đầu quét từ khối = %d", blockNumber)
	log.Printf("TransferSignature: %s", chainData.Config.TransferSignature)
	log.Printf("==================================")

	blockCounter := 0
	rpcClient := client.Client()

	for {
		blockCounter++

		// Mỗi 50 khối, kiểm tra xem có bị bỏ lỡ khối nào không
		if blockCounter%50 == 0 {
			var latestBlockHex string
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := rpcClient.CallContext(ctx, &latestBlockHex, "eth_blockNumber")
			cancel()

			if err == nil {
				latestBlock := new(big.Int)
				latestBlock.SetString(latestBlockHex[2:], 16)

				gap := new(big.Int).Sub(latestBlock, blockNumber)
				if gap.Cmp(big.NewInt(20)) > 0 {
					// Có khoảng cách lớn, xử lý nhanh các khối bị bỏ lỡ
					nextBatchEnd := new(big.Int).Add(blockNumber, big.NewInt(100))
					if nextBatchEnd.Cmp(latestBlock) > 0 {
						nextBatchEnd.Set(latestBlock)
					}

					log.Printf("⚠️ Phát hiện khoảng cách %d khối trên chain %s. Đang quét nhanh từ %d đến %d...",
						gap, chainName, blockNumber, nextBatchEnd)

					for i := new(big.Int).Set(blockNumber); i.Cmp(nextBatchEnd) <= 0; i.Add(i, big.NewInt(1)) {
						if err := processBlock(rpcClient, i, chainName); err != nil {
							log.Printf("Lỗi khi xử lý khối %d: %v, tiếp tục...", i, err)
						}
						time.Sleep(50 * time.Millisecond)
					}

					blockNumber.Set(nextBatchEnd)
					blockNumber.Add(blockNumber, big.NewInt(1))

					processLock.Lock()
					chainData.LastProcessedBlock.Set(nextBatchEnd)
					processLock.Unlock()

					log.Printf("✅ Đã quét nhanh đến khối %d", nextBatchEnd)
					continue
				}
			}
		}

		// Xử lý khối tiếp theo
		err := processBlock(rpcClient, blockNumber, chainName)
		if err == nil {
			blockNumber = new(big.Int).Add(blockNumber, big.NewInt(1))

			processLock.Lock()
			if blockNumber.Cmp(chainData.LastProcessedBlock) > 0 {
				chainData.LastProcessedBlock = new(big.Int).Set(blockNumber)
				chainData.LastProcessedBlock.Sub(chainData.LastProcessedBlock, big.NewInt(1))
			}
			processLock.Unlock()

			time.Sleep(200 * time.Millisecond)
		} else {
			sleepTime := time.Duration(chainData.Config.TimeNeedToBlock) * time.Millisecond
			if sleepTime < 500*time.Millisecond {
				sleepTime = 500 * time.Millisecond
			}
			log.Printf("Đợi %v trước khi thử lại khối %d", sleepTime, blockNumber)
			time.Sleep(sleepTime)
		}
	}
}

func reverseHandleHTTP(client *ethclient.Client, chainName string, pastDuration time.Duration) {
	chainData := GetChainData(chainName)
	if chainData == nil {
		log.Fatalf("Không tìm thấy dữ liệu cho chain %s", chainName)
	}

	// Kết nối đến RPC client
	rpcClient := client.Client()

	// Lấy block hiện tại để bắt đầu quét ngược
	var latestBlockHex string
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := rpcClient.CallContext(ctx, &latestBlockHex, "eth_blockNumber")
	cancel()
	if err != nil {
		log.Fatalf("Lỗi khi lấy block hiện tại: %v", err)
	}

	// Chuyển đổi từ hex sang big.Int
	blockNumber := new(big.Int)
	blockNumber.SetString(latestBlockHex[2:], 16)

	// Tính thời gian mục tiêu
	targetTime := time.Now().Add(-pastDuration)

	log.Printf("======= KHỞI ĐỘNG HỆ THỐNG NGƯỢC CHO CHAIN %s =======", chainName)
	log.Printf("Bắt đầu quét từ khối hiện tại = %d", blockNumber)
	log.Printf("Thời gian mục tiêu: %v", targetTime)
	log.Printf("TransferSignature: %s", chainData.Config.TransferSignature)
	log.Printf("==================================")

	blockCounter := 0

	for {
		
		blockCounter++

		// Xử lý khối hiện tại
		err := processBlock(rpcClient, blockNumber, chainName)
		if err == nil {
			// Lấy thời gian của block để kiểm tra điều kiện dừng
			var block map[string]interface{}
			blockHex := fmt.Sprintf("0x%x", blockNumber)

			if err := rpcClient.Call(&block, "eth_getBlockByNumber", blockHex, false); err == nil {
				fmt.Println("Block:", block)
				// Lấy thời gian của block
				blockTimeHex, _ := block["timestamp"].(string)
				blockTimeInt := new(big.Int)
				if len(blockTimeHex) > 2 {
					blockTimeInt.SetString(blockTimeHex[2:], 16)
				}
				blockTime := time.Unix(blockTimeInt.Int64(), 0)
				
				// Kiểm tra nếu đã đạt đến thời gian mục tiêu
				if blockTime.Before(targetTime) || blockTime.Equal(targetTime) {
					log.Printf("✅ Đã đạt đến thời gian mục tiêu tại khối %d", blockNumber)
					break
				}
			}
			// Giảm số block để quét ngược về quá khứ
			blockNumber = new(big.Int).Sub(blockNumber, big.NewInt(1))

			// Cập nhật LastProcessedBlock
			processLock.Lock()
			chainData.LastProcessedBlock = new(big.Int).Set(blockNumber)
			chainData.LastProcessedBlock.Add(chainData.LastProcessedBlock, big.NewInt(1))
			processLock.Unlock()

			// Mỗi 50 khối, in thông tin tiến độ
			if blockCounter%50 == 0 {
				log.Printf("🔄 Đã quét ngược %d khối, hiện tại ở khối %d", blockCounter, blockNumber)
			}

			time.Sleep(200 * time.Millisecond)
		} else {
			sleepTime := time.Duration(chainData.Config.TimeNeedToBlock) * time.Millisecond
			if sleepTime < 500*time.Millisecond {
				sleepTime = 500 * time.Millisecond
			}
			log.Printf("Đợi %v trước khi thử lại khối %d", sleepTime, blockNumber)
			time.Sleep(sleepTime)
		}
	}

	log.Printf("✅ Hoàn thành xử lý các block ngược cho chain %s", chainName)
}

// Xử lý chain qua HTTP
func handle_chain_http(chainName string) {
	chainData := InitChainData(chainName)
	load_config(chooseChain[chainName], chainName)

	client, err := ethclient.Dial(chainData.Config.WssRPC)
	if err != nil {
		log.Fatalf("Không thể kết nối đến RPC cho %s: %v", chainName, err)
	}
	defer client.Close()
	pastDuration := 1 * time.Hour

	// reverseHandleHTTP(client, chainName, pastDuration)
	reverseHandleHTTP(client, chainName, pastDuration)

	// continueHandleCosmosHTTP(chainName)
}

func handle_http() {
	// Trước tiên, nhập dữ liệu từ file log vào database (nếu có)
	go importAllLogFiles()

	// Sau đó tiếp tục với các chain
	chains := []string{
		"ethereum",
		"bsc",
		"avalanche",
		"polygon",
		"arbitrum",
		"optimism",
		"fantom",
		"base",
	}

	for _, chain := range chains {
		go handle_chain_http(chain)
	}

	// Giữ cho chương trình chạy mãi
	select {}
}
