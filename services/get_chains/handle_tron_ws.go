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

	"github.com/gorilla/websocket"
)

// Logger cho việc ghi log vào file
var tronWSFileLogger *log.Logger

// Thiết lập logger cho WebSocket Tron
func setupTronWSLogger() {
	// Tạo thư mục logs nếu chưa tồn tại
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		err := os.Mkdir("logs", 0755)
		if err != nil {
			log.Fatalf("Không thể tạo thư mục logs: %v", err)
		}
	}

	// Tạo file log với tên bao gồm timestamp
	timeStr := time.Now().Format("2006-01-02_15-04-05")
	logFile, err := os.Create(fmt.Sprintf("logs/tron_ws_%s.log", timeStr))
	if err != nil {
		log.Fatalf("Không thể tạo file log: %v", err)
	}

	// Thiết lập logger
	tronWSFileLogger = log.New(logFile, "", log.LstdFlags)
	tronWSFileLogger.Println("===== BẮT ĐẦU THEO DÕI BLOCKCHAIN TRON THEO THỜI GIAN THỰC =====")
}

// Hàm chính để xử lý WebSocket Tron
func handle_tron_ws() {
	// Thiết lập logger
	setupTronWSLogger()

	log.Println("Bắt đầu theo dõi blockchain Tron theo thời gian thực...")
	tronWSFileLogger.Println("Bắt đầu theo dõi blockchain Tron theo thời gian thực...")

	// Danh sách các node RPC để thử kết nối
	tronNodes := []string{
		"https://tron-rpc.publicnode.com/jsonrpc",
		"https://api.trongrid.io/jsonrpc",
		"https://rpc.ankr.com/tron",
	}

	// Chọn node đầu tiên làm mặc định
	primaryNodeURL := tronNodes[0]
	log.Printf("Sử dụng node Tron chính: %s", primaryNodeURL)
	tronWSFileLogger.Printf("Sử dụng node Tron chính: %s", primaryNodeURL)

	// Lấy khối mới nhất để bắt đầu
	latestBlock, err := getTronLatestBlockWS(primaryNodeURL)
	if err != nil {
		// Thử các node khác nếu node chính không hoạt động
		for i := 1; i < len(tronNodes); i++ {
			log.Printf("Thử kết nối đến node dự phòng: %s", tronNodes[i])
			latestBlock, err = getTronLatestBlockWS(tronNodes[i])
			if err == nil {
				primaryNodeURL = tronNodes[i]
				log.Printf("Chuyển sang sử dụng node: %s", primaryNodeURL)
				tronWSFileLogger.Printf("Chuyển sang sử dụng node: %s", primaryNodeURL)
				break
			}
		}

		if err != nil {
			log.Fatalf("Không thể kết nối đến bất kỳ node Tron nào: %v", err)
		}
	}

	log.Printf("Khối Tron mới nhất: #%d", latestBlock.Number)
	tronWSFileLogger.Printf("===== BẮT ĐẦU THEO DÕI TỪ KHỐI #%d =====", latestBlock.Number)

	// Lưu số khối cuối cùng đã xử lý
	lastProcessedBlock := latestBlock.Number

	// Khoảng thời gian giữa các lần kiểm tra khối mới (3 giây)
	pollInterval := 3 * time.Second

	// Vòng lặp chính để theo dõi các khối mới
	for {
		// Lấy khối mới nhất hiện tại
		currentLatestBlock, err := getTronLatestBlockWS(primaryNodeURL)
		if err != nil {
			log.Printf("Lỗi khi lấy khối mới nhất: %v, thử lại sau %s", err, pollInterval)
			time.Sleep(pollInterval)
			continue
		}

		// Nếu có khối mới
		if currentLatestBlock.Number > lastProcessedBlock {
			log.Printf("Phát hiện %d khối mới (từ #%d đến #%d)",
				currentLatestBlock.Number-lastProcessedBlock,
				lastProcessedBlock+1,
				currentLatestBlock.Number)

			// Xử lý từng khối mới, từ cũ đến mới
			for blockNum := lastProcessedBlock + 1; blockNum <= currentLatestBlock.Number; blockNum++ {
				// Lấy thông tin chi tiết của khối
				block, err := getTronBlockByNumWS(blockNum, primaryNodeURL)
				if err != nil {
					log.Printf("Lỗi khi lấy khối #%d: %v", blockNum, err)
					continue
				}

				// Xử lý khối
				processTronBlockWS(block, primaryNodeURL)

				// Cập nhật khối cuối cùng đã xử lý
				lastProcessedBlock = blockNum
			}
		} else {
			log.Printf("Không có khối mới. Khối mới nhất vẫn là #%d", lastProcessedBlock)
		}

		// Chờ đến lần kiểm tra tiếp theo
		time.Sleep(pollInterval)
	}
}

// Lấy khối mới nhất của Tron (phiên bản WebSocket)
func getTronLatestBlockWS(nodeURL string) (*TronBlock, error) {
	// Chuẩn bị yêu cầu JSON-RPC
	jsonData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_blockNumber",
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
	return getTronBlockByNumWS(blockNum, nodeURL)
}

// Lấy khối theo số (phiên bản WebSocket)
func getTronBlockByNumWS(blockNum uint64, nodeURL string) (*TronBlock, error) {
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

// Xử lý khối Tron (phiên bản WebSocket)
func processTronBlockWS(block *TronBlock, nodeURL string) {
	// Log thông tin khối
	log.Printf("Đang xử lý khối Tron #%d (hash: %s) với %d giao dịch",
		block.Number, block.BlockID, block.TransactionsCount)
	tronWSFileLogger.Printf("Đang xử lý khối Tron #%d (hash: %s) với %d giao dịch",
		block.Number, block.BlockID, block.TransactionsCount)

	// Chuyển đổi timestamp thành thời gian có thể đọc được
	blockTime := time.Unix(int64(block.Timestamp/1000), 0)
	formattedTime := blockTime.Format("2006-01-02 15:04:05")

	// Log thông tin thời gian khối
	log.Printf("Khối #%d được tạo vào: %s", block.Number, formattedTime)
	tronWSFileLogger.Printf("Khối #%d được tạo vào: %s", block.Number, formattedTime)

	// Xử lý từng giao dịch trong khối
	for i, tx := range block.Transactions {
		// Kiểm tra xem giao dịch có thành công không
		isSuccess := false
		if len(tx.Ret) > 0 {
			isSuccess = tx.Ret[0].ContractRet == "SUCCESS"
		}

		// Bỏ qua các giao dịch không thành công
		if !isSuccess {
			continue
		}

		// Xử lý từng hợp đồng trong giao dịch
		for _, contract := range tx.RawData.Contract {
			// Phân tích loại hợp đồng
			switch contract.Type {
			case "TransferContract":
				// Xử lý chuyển TRX
				processTransferContract(tx.TxID, contract, block.Number, formattedTime)
			case "TransferAssetContract":
				// Xử lý chuyển token TRC10
				processTransferAssetContract(tx.TxID, contract, block.Number, formattedTime)
			case "TriggerSmartContract":
				// Xử lý hợp đồng thông minh (có thể bao gồm chuyển TRC20)
				processTriggerSmartContract(tx.TxID, contract, block.Number, formattedTime, nodeURL)
			default:
				// Log các loại hợp đồng khác
				log.Printf("Giao dịch #%d trong khối #%d: Loại hợp đồng không được xử lý: %s",
					i, block.Number, contract.Type)
			}
		}
	}

	// Log khi hoàn thành xử lý khối
	log.Printf("Đã xử lý xong khối Tron #%d", block.Number)
	tronWSFileLogger.Printf("Đã xử lý xong khối Tron #%d", block.Number)
}

// Xử lý hợp đồng chuyển TRX
func processTransferContract(txID string, contract Contract, blockNum uint64, blockTime string) {
	// Phân tích tham số hợp đồng
	var params struct {
		From  string `json:"from"`
		To    string `json:"to"`
		Value string `json:"value"`
	}

	if err := json.Unmarshal(contract.Parameter.Value, &params); err != nil {
		log.Printf("Lỗi khi phân tích tham số TransferContract: %v", err)
		return
	}

	// Chuyển đổi địa chỉ sang định dạng base58 nếu cần
	fromAddress := params.From
	toAddress := params.To

	// Chuyển đổi giá trị
	amount := params.Value
	amountFormatted := formatAmount(amount)

	// Log thông tin giao dịch
	log.Printf("Giao dịch TRX: %s, Từ: %s, Đến: %s, Số lượng: %s TRX, Khối: #%d, Thời gian: %s",
		txID, fromAddress, toAddress, amountFormatted, blockNum, blockTime)
	tronWSFileLogger.Printf("Giao dịch TRX: %s, Từ: %s, Đến: %s, Số lượng: %s TRX, Khối: #%d, Thời gian: %s",
		txID, fromAddress, toAddress, amountFormatted, blockNum, blockTime)
}

// Xử lý hợp đồng chuyển token TRC10
func processTransferAssetContract(txID string, contract Contract, blockNum uint64, blockTime string) {
	// Phân tích tham số hợp đồng
	var params struct {
		AssetName string `json:"asset_name"`
		From      string `json:"from"`
		To        string `json:"to"`
		Amount    string `json:"amount"`
	}

	if err := json.Unmarshal(contract.Parameter.Value, &params); err != nil {
		log.Printf("Lỗi khi phân tích tham số TransferAssetContract: %v", err)
		return
	}

	// Chuyển đổi địa chỉ sang định dạng base58 nếu cần
	fromAddress := params.From
	toAddress := params.To

	// Chuyển đổi giá trị
	amountFormatted := formatAmount(params.Amount)

	// Log thông tin giao dịch
	log.Printf("Giao dịch TRC10: %s, Token: %s, Từ: %s, Đến: %s, Số lượng: %s, Khối: #%d, Thời gian: %s",
		txID, params.AssetName, fromAddress, toAddress, amountFormatted, blockNum, blockTime)
	tronWSFileLogger.Printf("Giao dịch TRC10: %s, Token: %s, Từ: %s, Đến: %s, Số lượng: %s, Khối: #%d, Thời gian: %s",
		txID, params.AssetName, fromAddress, toAddress, amountFormatted, blockNum, blockTime)
}

// Xử lý hợp đồng thông minh (có thể bao gồm chuyển TRC20)
func processTriggerSmartContract(txID string, contract Contract, blockNum uint64, blockTime string, nodeURL string) {
	// Phân tích tham số hợp đồng
	var params struct {
		ContractAddress string `json:"contract_address"`
		Data            string `json:"data"`
		From            string `json:"from"`
		To              string `json:"to"`
	}

	if err := json.Unmarshal(contract.Parameter.Value, &params); err != nil {
		log.Printf("Lỗi khi phân tích tham số TriggerSmartContract: %v", err)
		return
	}

	// Kiểm tra xem có phải là giao dịch chuyển TRC20 không
	if len(params.Data) >= 8 && params.Data[:8] == "a9059cbb" {
		// Đây có thể là hàm transfer() của TRC20
		// Phân tích dữ liệu để lấy địa chỉ người nhận và số lượng
		if len(params.Data) >= 136 {
			// Log thông tin giao dịch TRC20
			log.Printf("Giao dịch TRC20: %s, Hợp đồng: %s, Từ: %s, Dữ liệu: %s, Khối: #%d, Thời gian: %s",
				txID, params.ContractAddress, params.From, params.Data, blockNum, blockTime)
			tronWSFileLogger.Printf("Giao dịch TRC20: %s, Hợp đồng: %s, Từ: %s, Dữ liệu: %s, Khối: #%d, Thời gian: %s",
				txID, params.ContractAddress, params.From, params.Data, blockNum, blockTime)
		}
	} else {
		// Log các giao dịch hợp đồng thông minh khác
		log.Printf("Giao dịch hợp đồng thông minh: %s, Hợp đồng: %s, Từ: %s, Khối: #%d, Thời gian: %s",
			txID, params.ContractAddress, params.From, blockNum, blockTime)
		tronWSFileLogger.Printf("Giao dịch hợp đồng thông minh: %s, Hợp đồng: %s, Từ: %s, Khối: #%d, Thời gian: %s",
			txID, params.ContractAddress, params.From, blockNum, blockTime)
	}
}

// Hàm định dạng số lượng token
func formatAmount(value string) string {
	// Chuyển đổi từ hex sang decimal nếu cần
	if strings.HasPrefix(value, "0x") {
		decValue := hexToUint64(value)
		value = fmt.Sprintf("%d", decValue)
	}

	// Thêm dấu phẩy ngăn cách hàng nghìn
	var result strings.Builder
	length := len(value)

	for i, char := range value {
		result.WriteRune(char)
		if (length-i-1)%3 == 0 && i < length-1 {
			result.WriteRune(',')
		}
	}

	return result.String()
}

// Kiểm tra xem có hỗ trợ WebSocket không
func checkTronWebSocketSupport() bool {
	// Danh sách các endpoint WebSocket tiềm năng của Tron
	potentialWSEndpoints := []string{
		"wss://api.trongrid.io/v1/ws",
		"wss://mainnet.trongrid.io/ws",
		"wss://tron-ws.publicnode.com",
	}

	for _, endpoint := range potentialWSEndpoints {
		log.Printf("Kiểm tra endpoint WebSocket: %s", endpoint)

		// Thiết lập header
		header := http.Header{}
		header.Add("Origin", "https://trongrid.io")

		// Thử kết nối
		dialer := websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
		}

		c, _, err := dialer.Dial(endpoint, header)
		if err == nil {
			log.Printf("Kết nối WebSocket thành công đến %s", endpoint)
			c.Close()
			return true
		}

		log.Printf("Không thể kết nối đến %s: %v", endpoint, err)
	}

	log.Println("Không tìm thấy endpoint WebSocket nào hoạt động cho Tron")
	return false
}

// Hàm chính để xử lý Tron theo thời gian thực
func handle_tron_realtime() {
	// Kiểm tra xem có hỗ trợ WebSocket không
	if checkTronWebSocketSupport() {
		log.Println("Đã tìm thấy hỗ trợ WebSocket cho Tron, sử dụng phương thức WebSocket")
		// Nếu có hỗ trợ WebSocket, sử dụng phương thức WebSocket
		// Hiện tại chưa có triển khai cụ thể vì Tron không có API WebSocket chính thức
		// Đây là nơi để thêm code xử lý WebSocket trong tương lai
	} else {
		log.Println("Không tìm thấy hỗ trợ WebSocket cho Tron, sử dụng phương thức polling")
		// Nếu không có hỗ trợ WebSocket, sử dụng phương thức polling
		handle_tron_ws()
	}
}

// Hàm để xử lý các khối Tron theo thời gian thực với khả năng phục hồi
func handle_tron_ws_resilient() {
	// Thiết lập logger
	setupTronWSLogger()

	log.Println("Bắt đầu xử lý Tron theo thời gian thực với khả năng phục hồi")
	tronWSFileLogger.Println("Bắt đầu xử lý Tron theo thời gian thực với khả năng phục hồi")

	// Danh sách các node RPC để thử kết nối
	tronNodes := []string{
		"https://tron-rpc.publicnode.com/jsonrpc",
		"https://api.trongrid.io/jsonrpc",
		"https://rpc.ankr.com/tron",
	}

	// Chọn node đầu tiên làm mặc định
	primaryNodeURL := tronNodes[0]
	log.Printf("Sử dụng node Tron chính: %s", primaryNodeURL)
	tronWSFileLogger.Printf("Sử dụng node Tron chính: %s", primaryNodeURL)

	// Lấy khối mới nhất để bắt đầu
	var lastProcessedBlock uint64 = 0
	var err error

	// Thử lấy khối mới nhất từ tất cả các node
	for _, nodeURL := range tronNodes {
		log.Printf("Thử lấy khối mới nhất từ node: %s", nodeURL)
		latestBlock, err := getTronLatestBlockWS(nodeURL)
		if err == nil {
			primaryNodeURL = nodeURL
			lastProcessedBlock = latestBlock.Number
			log.Printf("Sử dụng node: %s, khối mới nhất: #%d", primaryNodeURL, lastProcessedBlock)
			tronWSFileLogger.Printf("Sử dụng node: %s, khối mới nhất: #%d", primaryNodeURL, lastProcessedBlock)
			break
		}
		log.Printf("Không thể lấy khối mới nhất từ %s: %v", nodeURL, err)
	}

	if lastProcessedBlock == 0 {
		log.Fatalf("Không thể kết nối đến bất kỳ node Tron nào")
	}

	// Khoảng thời gian giữa các lần kiểm tra khối mới (3 giây)
	pollInterval := 3 * time.Second

	// Vòng lặp chính để theo dõi các khối mới
	for {
		// Lấy khối mới nhất hiện tại
		var currentLatestBlock *TronBlock
		var nodeWorking bool = false

		// Thử tất cả các node cho đến khi tìm thấy một node hoạt động
		for _, nodeURL := range tronNodes {
			currentLatestBlock, err = getTronLatestBlockWS(nodeURL)
			if err == nil {
				if nodeURL != primaryNodeURL {
					log.Printf("Chuyển sang sử dụng node: %s", nodeURL)
					tronWSFileLogger.Printf("Chuyển sang sử dụng node: %s", nodeURL)
					primaryNodeURL = nodeURL
				}
				nodeWorking = true
				break
			}
		}

		if !nodeWorking {
			log.Printf("Tất cả các node đều không hoạt động, thử lại sau %s", pollInterval)
			time.Sleep(pollInterval)
			continue
		}

		// Nếu có khối mới
		if currentLatestBlock.Number > lastProcessedBlock {
			log.Printf("Phát hiện %d khối mới (từ #%d đến #%d)",
				currentLatestBlock.Number-lastProcessedBlock,
				lastProcessedBlock+1,
				currentLatestBlock.Number)

			// Xử lý từng khối mới, từ cũ đến mới
			for blockNum := lastProcessedBlock + 1; blockNum <= currentLatestBlock.Number; blockNum++ {
				// Lấy thông tin chi tiết của khối
				block, err := getTronBlockByNumWS(blockNum, primaryNodeURL)
				if err != nil {
					log.Printf("Lỗi khi lấy khối #%d: %v, thử node khác", blockNum, err)

					// Thử các node khác nếu node hiện tại không hoạt động
					var blockFound bool = false
					for _, nodeURL := range tronNodes {
						if nodeURL == primaryNodeURL {
							continue
						}

						block, err = getTronBlockByNumWS(blockNum, nodeURL)
						if err == nil {
							log.Printf("Lấy khối #%d thành công từ node: %s", blockNum, nodeURL)
							primaryNodeURL = nodeURL
							blockFound = true
							break
						}
					}

					if !blockFound {
						log.Printf("Không thể lấy khối #%d từ bất kỳ node nào, bỏ qua", blockNum)
						continue
					}
				}

				// Xử lý khối
				processTronBlockWS(block, primaryNodeURL)

				// Cập nhật khối cuối cùng đã xử lý
				lastProcessedBlock = blockNum
			}
		} else {
			log.Printf("Không có khối mới. Khối mới nhất vẫn là #%d", lastProcessedBlock)
		}

		// Chờ đến lần kiểm tra tiếp theo
		time.Sleep(pollInterval)
	}
}
