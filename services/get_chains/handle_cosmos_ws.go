package get_chains

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// Config cấu trúc để lưu trữ dữ liệu từ file config
type ConfigCosmos struct {
	RPC    string `json:"rpc"`
	WssRpc string `json:"wssRpc"`
	Chain  string `json:"chain"`
}

// WSRequest cấu trúc cho yêu cầu websocket
type WSRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	ID      int         `json:"id"`
	Params  interface{} `json:"params,omitempty"`
}

// TransactionRecord cấu trúc để lưu trữ thông tin giao dịch
type TransactionRecord struct {
	// Thông tin block
	BlockHeight string    `json:"block_height"`
	BlockHash   string    `json:"block_hash"`
	BlockTime   time.Time `json:"block_time"`
	ChainID     string    `json:"chain_id"`

	// Thông tin giao dịch
	TxHash         string `json:"tx_hash"`
	From           string `json:"from"`
	To             string `json:"to"`
	Amount         string `json:"amount"`
	Token          string `json:"token"`
	TotalAmount    string `json:"total_amount"`
	TxType         string `json:"tx_type"`
	Timestamp      string `json:"timestamp"`
	EventSignature string `json:"event_signature"`
	RawData        string `json:"raw_data"`
}

// Mutex để đồng bộ hóa ghi log
var logMutex sync.Mutex

func handle_cosmos_ws() {
	// Mở file log
	logFile, err := os.OpenFile("websocket.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Không thể mở file log: %v", err)
	}
	defer logFile.Close()

	// Chuyển hướng log vào file
	log.SetOutput(logFile)

	// Đọc file cấu hình
	configFile, err := os.ReadFile("./services/get_chains/config_chain/config-cosmos.json")
	if err != nil {
		log.Fatalf("Không thể đọc file cấu hình: %v", err)
	}

	// Parse cấu hình
	var ConfigCosmos ConfigCosmos
	if err := json.Unmarshal(configFile, &ConfigCosmos); err != nil {
		log.Fatalf("Không thể parse file cấu hình: %v", err)
	}

	log.Printf("Kết nối đến blockchain %s qua WebSocket: %s\n", ConfigCosmos.Chain, ConfigCosmos.WssRpc)

	// Kết nối WebSocket
	c, _, err := websocket.DefaultDialer.Dial(ConfigCosmos.WssRpc, nil)
	if err != nil {
		log.Fatalf("Không thể kết nối WebSocket: %v", err)
	}
	defer c.Close()

	// Đăng ký nhận sự kiện block mới và giao dịch
	subscribeMsg := WSRequest{
		JSONRPC: "2.0",
		Method:  "subscribe",
		ID:      1,
		Params:  map[string]string{"query": "tm.event='NewBlock'"},
	}

	if err := c.WriteJSON(subscribeMsg); err != nil {
		log.Fatalf("Không thể gửi yêu cầu đăng ký: %v", err)
	}

	// Xử lý tín hiệu để thoát chương trình một cách an toàn
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Nhận tín hiệu thoát, đang đóng kết nối...")
		cancel()
	}()

	log.Println("Đang lắng nghe các block mới...")

	// Lắng nghe các block mới
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, message, err := c.ReadMessage()
				if err != nil {
					log.Printf("Lỗi khi đọc tin nhắn: %v", err)
					time.Sleep(time.Second)
					continue
				}

				// Xử lý thông điệp block
				processBlockData(message, ConfigCosmos.Chain)
			}
		}
	}()

	<-ctx.Done()
	log.Println("Chương trình đã kết thúc")
}

// Xử lý dữ liệu block nhận được
func processBlockData(message []byte, chainName string) {
	// Parse JSON dữ liệu
	var wsResponse struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			Data struct {
				Type  string `json:"type"`
				Value struct {
					Block struct {
						Data struct {
							Txs []string `json:"txs"` // Giao dịch dạng base64
						} `json:"data"`
						Header struct {
							Height   string `json:"height"`
							ChainID  string `json:"chain_id"`
							Time     string `json:"time"`
							AppHash  string `json:"app_hash"`
							DataHash string `json:"data_hash"`
						} `json:"header"`
					} `json:"block"`
					BlockID struct {
						Hash string `json:"hash"`
					} `json:"block_id"`
				} `json:"value"`
			} `json:"data"`
			Events struct {
				TransferAmount    []string `json:"transfer.amount"`
				TransferRecipient []string `json:"transfer.recipient"`
				TransferSender    []string `json:"transfer.sender"`
				MessageSender     []string `json:"message.sender"`
				MessageMode       []string `json:"message.mode"`
				TMEvent           []string `json:"tm.event"`
			} `json:"events"`
		} `json:"result"`
	}

	if err := json.Unmarshal(message, &wsResponse); err != nil {
		log.Printf("Lỗi khi parse dữ liệu JSON: %v", err)
		return
	}

	// Kiểm tra xem đây có phải là sự kiện NewBlock
	if wsResponse.Result.Data.Type != "tendermint/event/NewBlock" {
		return
	}

	// Trích xuất thông tin cơ bản của block
	blockData := wsResponse.Result.Data.Value
	blockHeader := blockData.Block.Header

	blockHeight := blockHeader.Height
	blockHash := blockData.BlockID.Hash
	blockTime, _ := time.Parse(time.RFC3339, blockHeader.Time)
	chainID := blockHeader.ChainID

	logMutex.Lock()
	log.Printf("BLOCK MỚI #%s | Chain: %s | Time: %s | Hash: %s",
		blockHeight, chainID, blockTime.Format(time.RFC3339), blockHash)
	logMutex.Unlock()

	// Lấy danh sách các giao dịch
	txs := blockData.Block.Data.Txs
	if len(txs) == 0 {
		logMutex.Lock()
		log.Printf("Block #%s không có giao dịch", blockHeight)
		logMutex.Unlock()
		return
	}

	logMutex.Lock()
	log.Printf("Số lượng giao dịch trong block #%s: %d", blockHeight, len(txs))
	logMutex.Unlock()

	// Lấy thông tin về các sự kiện transfer
	transferAmounts := wsResponse.Result.Events.TransferAmount
	transferRecipients := wsResponse.Result.Events.TransferRecipient
	transferSenders := wsResponse.Result.Events.TransferSender

	// Lấy thông tin về các sự kiện message
	messageSenders := wsResponse.Result.Events.MessageSender
	messageModes := wsResponse.Result.Events.MessageMode

	// Tạo danh sách các giao dịch
	var transactions []TransactionRecord
	for i, txBase64 := range txs {
		// Giải mã giao dịch từ base64
		txBytes, err := base64.StdEncoding.DecodeString(txBase64)
		if err != nil {
			continue
		}

		// Tính hash giao dịch
		hash := sha256.Sum256(txBytes)
		txHash := fmt.Sprintf("%x", hash[:])

		// Lấy thông tin về từ/đến/số lượng từ events ở cấp cao nhất
		var from, to, amount, token string
		var totalAmount string
		var txType string
		var eventSignature string

		// Đặt giá trị mặc định là "Unknown" hoặc chuỗi rỗng
		from = ""
		to = ""
		totalAmount = ""
		txType = "Unknown"
		eventSignature = ""

		// Lấy thông tin nếu có
		if i < len(transferSenders) {
			from = transferSenders[i]
		}

		if i < len(transferRecipients) {
			to = transferRecipients[i]
		}

		if i < len(transferAmounts) {
			totalAmount = transferAmounts[i]
			amount, token = parseAmount(totalAmount)
		}

		if i < len(messageModes) {
			txType = messageModes[i]
		}

		// Tạo event signature đơn giản
		if txType != "Unknown" {
			eventSignature = txType
			if i < len(messageSenders) {
				eventSignature = fmt.Sprintf("%s.%s", txType, messageSenders[i])
			}
		}

		// Tạo raw data từ giao dịch gốc
		rawData := base64.StdEncoding.EncodeToString(txBytes)

		// Tạo record với các thông tin đã xác định
		record := TransactionRecord{
			BlockHeight:    blockHeight,
			BlockHash:      blockHash,
			BlockTime:      blockTime,
			ChainID:        chainID,
			TxHash:         txHash,
			From:           from,
			To:             to,
			Amount:         amount,
			Token:          token,
			TotalAmount:    totalAmount,
			TxType:         txType,
			Timestamp:      blockHeader.Time,
			EventSignature: eventSignature,
			RawData:        rawData,
		}

		transactions = append(transactions, record)
	}

	// Ghi log và lưu các giao dịch vào database
	for _, record := range transactions {
		recordJSON, _ := json.MarshalIndent(record, "", "  ")
		logMutex.Lock()
		log.Printf("GIAO DỊCH TRONG BLOCK #%s:\n%s", blockHeight, string(recordJSON))
		logMutex.Unlock()
	}
}

// Hàm phân tích chuỗi amount để tách số lượng và denom
func parseAmount(amountStr string) (amount string, denom string) {
	if amountStr == "" {
		return "", ""
	}

	re := regexp.MustCompile(`^(\d+)(\D+)`)
	matches := re.FindStringSubmatch(amountStr)
	if len(matches) == 3 {
		return matches[1], matches[2]
	}

	// Xử lý chuỗi thông thường nếu regex không khớp
	for i, c := range amountStr {
		if c < '0' || c > '9' {
			if i > 0 {
				return amountStr[:i], amountStr[i:]
			}
			break
		}
	}

	return amountStr, ""
}
