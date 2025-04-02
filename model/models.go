package models

import (
	"log"
	"math/big"
	"os"
	"sync"
	"time"
)

type Config struct {
	RPC                 string `json:"rpc"`
	WssRPC              string `json:"wssRpc"`
	WssToken            string `json:"wssToken"`
	TransferSignature   string `json:"transferSignature"`
	Chain               string `json:"chain"`
	EthContractAddress  string `json:"ethContractAddress"`
	UsdtContractAddress string `json:"usdtContractAddress"`
	UsdcContractAddress string `json:"usdcContractAddress"`
	WrappedBTCAddress   string `json:"wrappedBTCAddress"`
	TimeNeedToBlock     int    `json:"timeNeedToBlock"`
}

type ChainData struct {
	Config              Config
	LastProcessedBlock  *big.Int
	ProcessedTxs        map[string]bool
	DisconnectedChannel chan struct{}
	LogData             map[string]interface{}
	IsProcessingReorg   bool
}

type Transaction struct {
	ID              int64
	Timestamp       time.Time
	BlockNumber     uint64
	TxHash          string
	Address         string
	Amount          string
	RawData         string
	EventSignature  string
	FromAddress     string
	ToAddress       string
	NameChain       string
	TransactionType string
}

type WebSocketMessage struct {
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params"`
}

type WebSocketResponse struct {
	Action  string      `json:"action"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
}

// Cấu trúc dữ liệu để lưu trữ thông tin lệnh
type Order struct {
	Timestamp time.Time
	OrderID   string
	Symbol    string
	Side      string
	Amount    float64
	Price     float64
	Source    string // Thêm trường này để biết nguồn dữ liệu (Binance)
}

type BinanceFlowData struct {
	Timestamp time.Time
	Incoming  map[string]float64
	Outgoing  map[string]float64
	Balance   map[string]float64
}

type KrakenFlowData struct {
	Timestamp time.Time
	Incoming  map[string]float64
	Outgoing  map[string]float64
	Balance   map[string]float64
}

var chains = make(map[string]*ChainData)
var chainsLock sync.RWMutex

var chooseChain = map[string]string{
	"ethereum":  "./config_chain/config-ethereum.json",
	"bsc":       "./config_chain/config-bsc.json",
	"avalanche": "./config_chain/config-avalanche.json",
	"polygon":   "./config_chain/config-polygon.json",
	"arbitrum":  "./config_chain/config-arbitrum.json",
	"optimism":  "./config_chain/config-optimism.json",
	"fantom":    "./config_chain/config-fantom.json",
	"base":      "./config_chain/config-base.json",
	"cosmos":    "./config_chain/config-cosmos.json",
	"tron":      "./config_chain/config-tron.json",
}

var processLock sync.Mutex

// Khai báo logger cho models.go
var modelsLogger *log.Logger
var modelsLogFile *os.File

// Hàm khởi tạo logger cho models.go
// func initModelsLogger() error {
// 	// Tạo thư mục logs nếu chưa tồn tại
// 	dataDir := "./data_btc_continue"
// 	absPath, err := filepath.Abs(dataDir)
// 	if err != nil {
// 		log.Printf("Không thể chuyển đổi đường dẫn tương đối: %v", err)
// 		absPath = dataDir
// 	}

// 	err = os.MkdirAll(absPath, 0755)
// 	if err != nil {
// 		return fmt.Errorf("không thể tạo thư mục log: %v", err)
// 	}

// 	// Tạo tên file log
// 	timeStr := time.Now().Format("2006-01-02_15-04-05")
// 	fileName := filepath.Join(absPath, fmt.Sprintf("models_%s.log", timeStr))

// 	// Mở file để ghi log
// 	modelsLogFile, err = os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
// 	if err != nil {
// 		return fmt.Errorf("không thể mở file log: %v", err)
// 	}

// 	// Tạo logger mới ghi vào cả file và console
// 	multiWriter := io.MultiWriter(modelsLogFile, os.Stdout)
// 	modelsLogger = log.New(multiWriter, "[MODELS] ", log.LstdFlags)
// 	modelsLogger.Printf("=== KHỞI TẠO MODELS LOG ===")
// 	modelsLogger.Printf("File log models được tạo tại: %s", fileName)

// 	return nil
// }

// // Gọi hàm init để khởi tạo logger khi package được load
// func init() {
// 	if err := initModelsLogger(); err != nil {
// 		log.Printf("Lỗi khởi tạo models logger: %v", err)
// 	}
// }

// // Hàm để xử lý dữ liệu từ Binance
// func ProcessBinanceData() BinanceFlowData {
// 	// Khởi tạo FlowData với các map rỗng
// 	flowData := BinanceFlowData{
// 		Timestamp: time.Now(),
// 		Incoming:  make(map[string]float64),
// 		Outgoing:  make(map[string]float64),
// 		Balance:   make(map[string]float64),
// 	}

// 	// Lấy dữ liệu từ Binance
// 	binanceOrders := services.GetBinanceOrders()
// 	modelsLogger.Printf("Xử lý %d giao dịch từ Binance", len(binanceOrders))

// 	// Chỉ xử lý dữ liệu từ Binance
// 	for _, order := range binanceOrders {
// 		symbol := order.Symbol

// 		// Kiểm tra nếu map chưa có key này thì khởi tạo
// 		if _, exists := flowData.Incoming[symbol]; !exists {
// 			flowData.Incoming[symbol] = 0
// 		}
// 		if _, exists := flowData.Outgoing[symbol]; !exists {
// 			flowData.Outgoing[symbol] = 0
// 		}
// 		if _, exists := flowData.Balance[symbol]; !exists {
// 			flowData.Balance[symbol] = 0
// 		}

// 		// Cập nhật dữ liệu dựa trên loại lệnh
// 		if order.Side == "buy" {
// 			flowData.Incoming[symbol] += order.Amount
// 			flowData.Balance[symbol] += order.Amount
// 		} else if order.Side == "sell" {
// 			flowData.Outgoing[symbol] += order.Amount
// 			flowData.Balance[symbol] -= order.Amount
// 		}
// 	}

// 	return flowData
// }

// // Hàm để xử lý dữ liệu từ Kraken
// func ProcessKrakenData() KrakenFlowData {
// 	// Khởi tạo KrakenFlowData với các map rỗng
// 	flowData := KrakenFlowData{
// 		Timestamp: time.Now(),
// 		Incoming:  make(map[string]float64),
// 		Outgoing:  make(map[string]float64),
// 		Balance:   make(map[string]float64),
// 	}

// 	// Lấy dữ liệu từ Kraken
// 	krakenOrders := services.ProcessKrakenData()
// 	modelsLogger.Printf("Xử lý %d giao dịch từ Kraken", len(krakenOrders))

// 	// Xử lý dữ liệu từ Kraken
// 	for _, order := range krakenOrders {
// 		symbol := order.Symbol

// 		// Kiểm tra nếu map chưa có key này thì khởi tạo
// 		if _, exists := flowData.Incoming[symbol]; !exists {
// 			flowData.Incoming[symbol] = 0
// 		}
// 		if _, exists := flowData.Outgoing[symbol]; !exists {
// 			flowData.Outgoing[symbol] = 0
// 		}
// 		if _, exists := flowData.Balance[symbol]; !exists {
// 			flowData.Balance[symbol] = 0
// 		}

// 		// Cập nhật dữ liệu dựa trên loại lệnh
// 		if order.Side == "buy" {
// 			flowData.Incoming[symbol] += order.Amount
// 			flowData.Balance[symbol] += order.Amount
// 		} else if order.Side == "sell" {
// 			flowData.Outgoing[symbol] += order.Amount
// 			flowData.Balance[symbol] -= order.Amount
// 		}
// 	}

// 	return flowData
// }

// // Hàm lấy dữ liệu từ Binance
// func GetCombinedOrderData() []services.Order {
// 	modelsLogger.Println("Lấy dữ liệu từ Binance và Kraken")

// 	// Lấy dữ liệu từ Binance
// 	binanceOrders := services.GetBinanceOrders()
// 	krakenOrders := services.ProcessKrakenData()

// 	modelsLogger.Printf("Đã lấy %d giao dịch Binance và %d giao dịch Kraken",
// 		len(binanceOrders), len(krakenOrders))

// 	// Trả về kết quả
// 	allOrders := append(binanceOrders, krakenOrders...)
// 	return allOrders
// }

// // Đóng file log khi chương trình kết thúc
// func CloseModelsLogger() {
// 	if modelsLogFile != nil {
// 		modelsLogger.Println("Đóng file log models")
// 		modelsLogFile.Close()
// 	}
// }
