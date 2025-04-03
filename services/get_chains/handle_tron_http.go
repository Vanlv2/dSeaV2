package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Logger cho việc ghi log vào file
var tronFileLogger *log.Logger

// Cấu trúc Block của Tron
type TronBlock struct {
	BlockID           string            `json:"blockID"`
	BlockHeader       BlockHeader       `json:"block_header"`
	Transactions      []TransactionTron `json:"transactions"`
	Number            uint64            // Số khối (được trích xuất từ BlockHeader)
	Timestamp         uint64            // Timestamp (được trích xuất từ BlockHeader)
	TransactionsCount int               // Số lượng giao dịch
}

// Cấu trúc header của khối Tron
type BlockHeader struct {
	RawData          RawData `json:"raw_data"`
	WitnessSignature string  `json:"witness_signature"`
}

type RawData struct {
	Number         uint64 `json:"number"`
	TxTrieRoot     string `json:"txTrieRoot"`
	WitnessAddress string `json:"witness_address"`
	ParentHash     string `json:"parentHash"`
	Version        int    `json:"version"`
	Timestamp      uint64 `json:"timestamp"`
}

// Cấu trúc giao dịch Tron
type TransactionTron struct {
	TxID       string    `json:"txID"`
	RawData    TxRawData `json:"raw_data"`
	Signatures []string  `json:"signature"`
	Ret        []TxRet   `json:"ret"`
}

type TxRawData struct {
	Contract      []Contract `json:"contract"`
	RefBlockBytes string     `json:"ref_block_bytes"`
	RefBlockHash  string     `json:"ref_block_hash"`
	Expiration    uint64     `json:"expiration"`
	Timestamp     uint64     `json:"timestamp"`
}

type Contract struct {
	Type      string            `json:"type"`
	Parameter ContractParameter `json:"parameter"`
}

type ContractParameter struct {
	Value   json.RawMessage `json:"value"`
	TypeURL string          `json:"type_url"`
}

type TxRet struct {
	ContractRet string `json:"contractRet"`
}

// TronTransactionLog - Cấu trúc log giao dịch Tron
type TronTransactionLog struct {
	BlockHeight     string `json:"block_height"`
	BlockHash       string `json:"block_hash"`
	BlockTime       string `json:"block_time"`
	ChainID         string `json:"chain_id"`
	TxHash          string `json:"tx_hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	ValueDecimal    string `json:"value_decimal"`
	ContractType    string `json:"contract_type"`
	Status          string `json:"status"`
	Energy          string `json:"energy"`
	EnergyUsed      string `json:"energy_used"`
	TxType          string `json:"tx_type"`
	Method          string `json:"method"`
	MethodSignature string `json:"method_signature"`
	Data            string `json:"data,omitempty"`
	Nonce           string `json:"nonce,omitempty"`
}

// Thiết lập logger
func setupTronLogger() {
	// Tạo thư mục logs nếu chưa tồn tại
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		err := os.Mkdir("logs", 0755)
		if err != nil {
			log.Fatalf("Không thể tạo thư mục logs: %v", err)
		}
	}

	// Tạo file log với tên bao gồm timestamp
	timeStr := time.Now().Format("2006-01-02_15-04-05")
	logFile, err := os.Create(fmt.Sprintf("logs/tron_scan_%s.log", timeStr))
	if err != nil {
		log.Fatalf("Không thể tạo file log: %v", err)
	}

	// Thiết lập logger
	tronFileLogger = log.New(logFile, "", log.LstdFlags)
	tronFileLogger.Println("===== BẮT ĐẦU QUÉT BLOCKCHAIN TRON =====")
}

// Hàm chính để quét các khối theo khoảng thời gian
func scanTronBlocksByTimeRange(duration time.Duration) error {
	// Thiết lập logger
	setupTronLogger()

	// Chỉ sử dụng một node duy nhất
	nodeURL := "https://tron-rpc.publicnode.com/jsonrpc"

	// Lấy khối mới nhất
	latestBlock, err := getTronLatestBlock(nodeURL)
	if err != nil {
		return fmt.Errorf("không thể lấy khối mới nhất: %v", err)
	}

	log.Printf("Khối Tron mới nhất: #%d", latestBlock.Number)
	tronFileLogger.Printf("===== BẮT ĐẦU QUÉT TỪ KHỐI #%d =====", latestBlock.Number)

	// Tính thời gian tối thiểu cần quét đến
	currentTime := time.Now()
	minTimeToScan := currentTime.Add(-duration)

	log.Printf("Quét từ: %s đến: %s", currentTime.Format(time.RFC3339), minTimeToScan.Format(time.RFC3339))
	tronFileLogger.Printf("Quét từ: %s đến: %s", currentTime.Format(time.RFC3339), minTimeToScan.Format(time.RFC3339))

	// Bắt đầu quét từ khối mới nhất và lùi dần
	currentBlockNum := latestBlock.Number
	processedBlocks := 0

	for {
		// Lấy thông tin khối hiện tại
		currentBlock, err := getTronBlockByNum(currentBlockNum, nodeURL)
		if err != nil {
			return fmt.Errorf("không thể lấy khối #%d: %v", currentBlockNum, err)
		}

		// Kiểm tra điều kiện dừng theo thời gian
		blockTime := time.Unix(int64(currentBlock.Timestamp/1000), 0)
		if blockTime.Before(minTimeToScan) {
			log.Printf("Đã đạt đến khối có thời gian trước %s, kết thúc quét", minTimeToScan.Format(time.RFC3339))
			tronFileLogger.Printf("Đã đạt đến khối có thời gian trước %s, kết thúc quét", minTimeToScan.Format(time.RFC3339))
			break
		}

		// Xử lý khối hiện tại
		processTronBlock(currentBlock, nodeURL)
		processedBlocks++

		// Nếu đã quét quá nhiều khối, có thể cân nhắc dừng để tránh quá tải
		if processedBlocks > 10000 {
			log.Printf("Đã quét %d khối, đạt giới hạn an toàn", processedBlocks)
			tronFileLogger.Printf("Đã quét %d khối, đạt giới hạn an toàn", processedBlocks)
			break
		}

		// Giảm số khối để lấy khối trước đó
		if currentBlockNum > 0 {
			currentBlockNum--
		} else {
			break
		}

		// Thêm độ trễ nhỏ để tránh quá tải API
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Hoàn thành quét %d khối", processedBlocks)
	tronFileLogger.Printf("===== KẾT THÚC QUÉT - ĐÃ XỬ LÝ %d KHỐI =====", processedBlocks)
	return nil
}

// Lấy khối mới nhất của Tron
func getTronLatestBlock(nodeURL string) (*TronBlock, error) {
	// Chuẩn bị yêu cầu JSON-RPC
	jsonData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_blockNumber", // Sử dụng phương thức Ethereum tương thích
		"params":  []interface{}{},
		"id":      1,
	}

	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi tạo yêu cầu JSON: %v", err)
	}

	// Gửi yêu cầu đến API
	req, err := http.NewRequest("POST", nodeURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("lỗi khi tạo yêu cầu HTTP: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi gửi yêu cầu HTTP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API trả về mã lỗi: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi đọc phản hồi: %v", err)
	}

	// Phân tích phản hồi JSON
	var result struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  string `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("lỗi khi phân tích JSON: %v", err)
	}

	// Chuyển đổi hex sang decimal
	blockNumHex := result.Result
	blockNum := hexToUint64(blockNumHex)

	// Lấy thông tin chi tiết của khối
	return getTronBlockByNum(blockNum, nodeURL)
}

// Lấy khối theo số
func getTronBlockByNum(blockNum uint64, nodeURL string) (*TronBlock, error) {
	// Tạo yêu cầu JSON-RPC
	blockNumHex := fmt.Sprintf("0x%x", blockNum)
	jsonData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBlockByNumber",
		"params":  []interface{}{blockNumHex, true}, // true để bao gồm đầy đủ thông tin giao dịch
		"id":      1,
	}

	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi tạo yêu cầu JSON: %v", err)
	}

	// Gửi yêu cầu đến API
	req, err := http.NewRequest("POST", nodeURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("lỗi khi tạo yêu cầu HTTP: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi gửi yêu cầu HTTP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API trả về mã lỗi: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi đọc phản hồi: %v", err)
	}

	// Phân tích phản hồi JSON
	var result struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("lỗi khi phân tích JSON: %v\n%s", err, string(body))
	}

	// Tạo đối tượng TronBlock từ dữ liệu nhận được
	var ethBlock struct {
		Hash         string        `json:"hash"`
		Number       string        `json:"number"`
		ParentHash   string        `json:"parentHash"`
		Timestamp    string        `json:"timestamp"`
		Transactions []interface{} `json:"transactions"`
	}

	if err := json.Unmarshal(result.Result, &ethBlock); err != nil {
		return nil, fmt.Errorf("lỗi khi phân tích dữ liệu khối: %v", err)
	}

	// Chuyển đổi dữ liệu sang định dạng TronBlock
	block := &TronBlock{
		BlockID: ethBlock.Hash,
		BlockHeader: BlockHeader{
			RawData: RawData{
				Number:     hexToUint64(ethBlock.Number),
				ParentHash: ethBlock.ParentHash,
				Timestamp:  hexToUint64(ethBlock.Timestamp) * 1000, // Chuyển đổi sang milliseconds
			},
		},
		Number:            hexToUint64(ethBlock.Number),
		Timestamp:         hexToUint64(ethBlock.Timestamp) * 1000,
		TransactionsCount: len(ethBlock.Transactions),
	}

	// Xử lý danh sách giao dịch
	for _, txRaw := range ethBlock.Transactions {
		txBytes, err := json.Marshal(txRaw)
		if err != nil {
			log.Printf("Lỗi khi chuyển đổi giao dịch: %v", err)
			continue
		}

		var ethTx struct {
			Hash     string `json:"hash"`
			From     string `json:"from"`
			To       string `json:"to"`
			Value    string `json:"value"`
			Gas      string `json:"gas"`
			GasPrice string `json:"gasPrice"`
			Input    string `json:"input"`
		}

		if err := json.Unmarshal(txBytes, &ethTx); err != nil {
			log.Printf("Lỗi khi phân tích giao dịch: %v", err)
			continue
		}

		// Tạo đối tượng Contract mô phỏng
		contract := Contract{
			Type: "TransferContract",
			Parameter: ContractParameter{
				Value:   []byte(`{"from":"` + ethTx.From + `","to":"` + ethTx.To + `","value":"` + ethTx.Value + `"}`),
				TypeURL: "type.googleapis.com/protocol.TransferContract",
			},
		}

		// Tạo giao dịch Tron từ dữ liệu Ethereum
		tx := TransactionTron{
			TxID: ethTx.Hash,
			RawData: TxRawData{
				Contract:   []Contract{contract},
				Timestamp:  block.Timestamp,
				Expiration: block.Timestamp + 60000, // Expiration thường là timestamp + 60 giây
			},
			Ret: []TxRet{
				{ContractRet: "SUCCESS"}, // Giả sử giao dịch thành công
			},
		}

		block.Transactions = append(block.Transactions, tx)
	}

	return block, nil
}

// Lấy chi tiết giao dịch
func getTronTransaction(txID string, nodeURL string) (*TransactionTron, error) {
	// Tạo yêu cầu JSON-RPC
	jsonData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getTransactionByHash",
		"params":  []interface{}{txID},
		"id":      1,
	}

	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi tạo yêu cầu JSON: %v", err)
	}

	// Gửi yêu cầu đến API
	req, err := http.NewRequest("POST", nodeURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("lỗi khi tạo yêu cầu HTTP: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi gửi yêu cầu HTTP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API trả về mã lỗi: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi đọc phản hồi: %v", err)
	}

	// Phân tích phản hồi JSON
	var result struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("lỗi khi phân tích JSON: %v", err)
	}

	// Tạo đối tượng Transaction từ dữ liệu
	var ethTx struct {
		Hash        string `json:"hash"`
		From        string `json:"from"`
		To          string `json:"to"`
		Value       string `json:"value"`
		Gas         string `json:"gas"`
		GasPrice    string `json:"gasPrice"`
		Input       string `json:"input"`
		BlockHash   string `json:"blockHash"`
		BlockNumber string `json:"blockNumber"`
	}

	if err := json.Unmarshal(result.Result, &ethTx); err != nil {
		return nil, fmt.Errorf("lỗi khi phân tích dữ liệu giao dịch: %v", err)
	}

	// Tạo đối tượng Contract
	contract := Contract{
		Type: determineContractType(ethTx.Input),
		Parameter: ContractParameter{
			Value:   []byte(`{"from":"` + ethTx.From + `","to":"` + ethTx.To + `","value":"` + ethTx.Value + `"}`),
			TypeURL: "type.googleapis.com/protocol.TransferContract",
		},
	}

	// Tạo giao dịch Tron
	blockNum := hexToUint64(ethTx.BlockNumber)
	timestamp := uint64(time.Now().UnixNano() / 1e6) // Mặc định thời gian hiện tại nếu không có

	// Lấy timestamp từ khối nếu có
	if ethTx.BlockNumber != "" && ethTx.BlockNumber != "0x0" {
		block, err := getTronBlockByNum(blockNum, nodeURL)
		if err == nil {
			timestamp = block.Timestamp
		}
	}

	tx := &TransactionTron{
		TxID: ethTx.Hash,
		RawData: TxRawData{
			Contract:   []Contract{contract},
			Timestamp:  timestamp,
			Expiration: timestamp + 60000, // Expiration thường là timestamp + 60 giây
		},
		Ret: []TxRet{
			{ContractRet: "SUCCESS"}, // Giả sử giao dịch thành công
		},
	}

	return tx, nil
}

// Xác định loại hợp đồng dựa trên dữ liệu đầu vào
func determineContractType(input string) string {
	if input == "0x" {
		return "TransferContract"
	}

	// Nếu có dữ liệu, có thể là TriggerSmartContract
	return "TriggerSmartContract"
}

// Xác định tên phương thức và chữ ký từ methodID
func getTronMethodName(methodID string) (string, string) {
	// Danh sách một số phương thức phổ biến trong Tron
	methodMap := map[string]string{
		"0xa9059cbb": "transfer(address,uint256)",
		"0x23b872dd": "transferFrom(address,address,uint256)",
		"0x095ea7b3": "approve(address,uint256)",
		"0x70a08231": "balanceOf(address)",
		"0x18160ddd": "totalSupply()",
		"0x313ce567": "decimals()",
		"0x06fdde03": "name()",
		"0x95d89b41": "symbol()",
		"0xdd62ed3e": "allowance(address,address)",
	}

	if signature, exists := methodMap[methodID]; exists {
		parts := strings.Split(signature, "(")
		return parts[0], signature
	}

	// Nếu không tìm thấy, trả về unknown
	return "unknown", "unknown"
}

// Chuyển đổi chuỗi hex thành uint64
func hexToUint64(hexStr string) uint64 {
	// Bỏ tiền tố "0x" nếu có
	if strings.HasPrefix(hexStr, "0x") {
		hexStr = hexStr[2:]
	}

	// Chuyển đổi chuỗi hex thành uint64
	var result uint64
	fmt.Sscanf(hexStr, "%x", &result)
	return result
}

// Chuyển đổi giá trị hex sang decimal value
func hexToDecimalTronValue(hexValue string) string {
	// Bỏ tiền tố "0x" nếu có
	if strings.HasPrefix(hexValue, "0x") {
		hexValue = hexValue[2:]
	}

	// Chuyển đổi chuỗi hex thành uint64
	var value uint64
	fmt.Sscanf(hexValue, "%x", &value)

	// Chuyển đổi sang chuỗi decimal
	return fmt.Sprintf("%d", value)
}

// Xác định loại giao dịch dựa trên tên phương thức
func determineTransactionTypehttpTron(methodName string) string {
	switch methodName {
	case "transfer":
		return "token_transfer"
	case "transferFrom":
		return "token_transfer_from"
	case "approve":
		return "token_approve"
	case "balanceOf", "totalSupply", "decimals", "name", "symbol", "allowance":
		return "token_view"
	default:
		if methodName == "unknown" {
			return "simple_transfer"
		}
		return "contract_call"
	}
}

// Xử lý thông tin khối và các giao dịch trong khối
func processTronBlock(block *TronBlock, nodeURL string) {
	txCount := len(block.Transactions)
	blockTime := time.Unix(int64(block.Timestamp/1000), 0).UTC()
	formattedBlockTime := blockTime.Format(time.RFC3339Nano)
	_ = blockTime.Format("2006-01-02 15:04:05")

	log.Printf("Đang xử lý khối Tron #%d, ID: %s, có %d giao dịch, thời gian: %s",
		block.Number, block.BlockID, txCount, formattedBlockTime)

	// Xử lý các giao dịch trong khối
	if txCount > 0 {
		for _, tx := range block.Transactions {
			tronFileLogger.Printf("\nGIAO DỊCH TRONG BLOCK #%d:", block.Number)

			// Xác định thời gian khối
			txBlockTime := formattedBlockTime

			// Xử lý từng hợp đồng trong giao dịch
			for i, contract := range tx.RawData.Contract {
				// Xác định loại giao dịch
				_ = contract.Type

				// Phân tích dữ liệu của hợp đồng
				var valueDecimal string = "0"
				var fromAddress string = ""
				var toAddress string = ""
				var dataInput string = ""

				// Giải mã dữ liệu hợp đồng dựa trên loại
				if contract.Type == "TransferContract" {
					var transferValue struct {
						From  string `json:"from"`
						To    string `json:"to"`
						Value string `json:"value"`
					}

					if err := json.Unmarshal(contract.Parameter.Value, &transferValue); err == nil {
						fromAddress = transferValue.From
						toAddress = transferValue.To
						valueDecimal = transferValue.Value
					}
				} else if contract.Type == "TriggerSmartContract" {
					var smartContract struct {
						From     string `json:"from"`
						Contract string `json:"contract_address"`
						Data     string `json:"data"`
					}

					if err := json.Unmarshal(contract.Parameter.Value, &smartContract); err == nil {
						fromAddress = smartContract.From
						toAddress = smartContract.Contract
						dataInput = smartContract.Data
					}
				}

				// Xác định method ID và tên phương thức
				methodID := ""
				if len(dataInput) >= 10 {
					methodID = dataInput[:10]
				}

				methodName, methodSignature := getTronMethodName(methodID)

				// Tạo raw_data JSON
				rawData := map[string]interface{}{
					"blockHash":   block.BlockID,
					"blockNumber": fmt.Sprintf("0x%x", block.Number),
					"hash":        tx.TxID,
					"from":        fromAddress,
					"to":          toAddress,
					"value":       valueDecimal,
					"input":       dataInput,
					"timestamp":   fmt.Sprintf("0x%x", block.Timestamp/1000),
				}
				_, _ = json.Marshal(rawData)

				// Tạo log giao dịch
				txLog := TronTransactionLog{
					BlockHeight:     fmt.Sprintf("%d", block.Number),
					BlockHash:       block.BlockID,
					BlockTime:       txBlockTime,
					ChainID:         "tron",
					TxHash:          tx.TxID,
					From:            fromAddress,
					To:              toAddress,
					Value:           valueDecimal,
					ValueDecimal:    valueDecimal,
					ContractType:    contract.Type,
					Status:          tx.Ret[0].ContractRet,
					TxType:          determineTransactionTypehttpTron(methodName),
					Method:          methodName,
					MethodSignature: methodSignature,
					Data:            dataInput,
				}

				// Định dạng JSON đẹp
				txJSON, _ := json.MarshalIndent(txLog, "", "    ")
				tronFileLogger.Println(string(txJSON))


				// Thêm log chi tiết cho contract nếu có data
				if dataInput != "" && dataInput != "0x" {
					contractDetail := map[string]interface{}{
						"index":            i + 1,
						"method_id":        methodID,
						"method_name":      methodName,
						"method_signature": methodSignature,
						"data_length":      len(dataInput),
						"contract_type":    contract.Type,
					}

					if len(dataInput) > 100 {
						contractDetail["data_preview"] = dataInput[:100] + "... (còn nữa)"
					} else {
						contractDetail["data"] = dataInput
					}

					contractJSON, _ := json.MarshalIndent(contractDetail, "", "    ")
					tronFileLogger.Printf("CHI TIẾT CONTRACT #%d:\n%s", i+1, string(contractJSON))
				}
			}
		}
	} else {
		tronFileLogger.Printf("\nKhối này không chứa giao dịch nào")
	}

	// Dòng phân cách giữa các khối
	tronFileLogger.Println("\n" + strings.Repeat("-", 80) + "\n")
}

func HandleTronHTTP() {
    log.Println("Bắt đầu quét blockchain Tron qua HTTP...")
    
    // Quét các khối trong 24 giờ qua
    duration := 24 * time.Hour
    
    // Tạo goroutine để quét không chặn luồng chính
    go func() {
        err := scanTronBlocksByTimeRange(duration)
        if err != nil {
            log.Printf("Lỗi khi quét blockchain Tron: %v", err)
        }
        
        // Thiết lập quét định kỳ (mỗi 1 giờ)
        ticker := time.NewTicker(1 * time.Hour)
        defer ticker.Stop()
        
        for range ticker.C {
            log.Println("Bắt đầu quét định kỳ blockchain Tron...")
            err := scanTronBlocksByTimeRange(1 * time.Hour) // Chỉ quét 1 giờ gần nhất
            if err != nil {
                log.Printf("Lỗi khi quét định kỳ blockchain Tron: %v", err)
            }
        }
    }()
    
    log.Println("Đã khởi động quét blockchain Tron qua HTTP")
}
