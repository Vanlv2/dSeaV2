package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// BTCOrder cấu trúc dữ liệu chung cho cả dữ liệu lịch sử và thời gian thực
type BTCOrder struct {
	Timestamp time.Time
	OrderID   string
	Symbol    string
	Side      string
	Amount    float64
	Price     float64
	Source    string // "RealTime" hoặc "Historical"
}

type BinanceConfig struct {
	WSURL                string        `json:"ws_url"`
	Tokens               []TokenConfig `json:"tokens"`
	PriceChangeThreshold float64       `json:"price_change_threshold"`
}

type TokenConfig struct {
	Symbol           string  `json:"symbol"`
	LargeOrderAmount float64 `json:"large_order_amount"`
	HugeOrderAmount  float64 `json:"huge_order_amount"`
}

type TradeEvent struct {
	Symbol    string `json:"s"`
	Price     string `json:"p"`
	Quantity  string `json:"q"`
	Timestamp int64  `json:"T"`
	IsBuyer   bool   `json:"m"`
}

type KlineEvent struct {
	Symbol string `json:"s"`
	Kline  struct {
		StartTime int64  `json:"t"`
		Close     string `json:"c"`
		Open      string `json:"o"`
		Interval  string `json:"i"`
		IsClosed  bool   `json:"x"`
	} `json:"k"`
}

// Biến global để lưu trữ dữ liệu thời gian thực
var realTimeOrders []BTCOrder
var realTimeOrdersMutex sync.RWMutex

// Logger cho từng loại tiền
var (
	btcLogger  *log.Logger
	ethLogger  *log.Logger
	solLogger  *log.Logger
	mainLogger *log.Logger
)

// Thiết lập logger cho từng loại tiền
func setupLoggers() error {
	// Tạo thư mục log nếu chưa tồn tại
	if _, err := os.Stat("./log"); os.IsNotExist(err) {
		if err := os.Mkdir("./log", 0755); err != nil {
			return fmt.Errorf("không thể tạo thư mục log: %v", err)
		}
	}

	// Mở file log chung
	mainLogFile, err := os.OpenFile("./services/get_chains/log/crypto_market.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("không thể mở file log chính: %v", err)
	}

	// Mở file log cho BTC
	btcLogFile, err := os.OpenFile("./services/get_chains/log/btc_market.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("không thể mở file log BTC: %v", err)
	}

	// Mở file log cho ETH
	ethLogFile, err := os.OpenFile("./services/get_chains/log/eth_market.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("không thể mở file log ETH: %v", err)
	}

	// Mở file log cho SOL
	solLogFile, err := os.OpenFile("./services/get_chains/log/sol_market.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("không thể mở file log SOL: %v", err)
	}

	// Khởi tạo các logger
	mainLogger = log.New(mainLogFile, "", log.LstdFlags)
	btcLogger = log.New(btcLogFile, "[BTC] ", log.LstdFlags)
	ethLogger = log.New(ethLogFile, "[ETH] ", log.LstdFlags)
	solLogger = log.New(solLogFile, "[SOL] ", log.LstdFlags)

	return nil
}

func LoadBinanceConfig(configFile string) (BinanceConfig, error) {
	var cfg BinanceConfig
	data, err := os.ReadFile(configFile)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func ConnectWebSocket(url string) (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	return conn, err
}

func Heartbeat(conn *websocket.Conn, name string) {
	for {
		time.Sleep(10 * time.Second)
		if err := conn.WriteMessage(websocket.PingMessage, []byte("ping-"+name)); err != nil {
			return
		}
	}
}

func ParseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// Lấy logger phù hợp cho từng loại tiền
func getLoggerForSymbol(symbol string) *log.Logger {
	switch {
	case strings.Contains(symbol, "BTC"):
		return btcLogger
	case strings.Contains(symbol, "ETH"):
		return ethLogger
	case strings.Contains(symbol, "SOL"):
		return solLogger
	default:
		return mainLogger
	}
}

func ProcessRealTimeMarketStream(cfg BinanceConfig, stopChan <-chan struct{}) {
	for _, token := range cfg.Tokens {
		// Sử dụng URL trực tiếp như trong file real_tIme_BTC.go
		streamURL := "wss://stream.binance.com:9443/ws/" + strings.ToLower(token.Symbol) + "@trade"

		mainLogger.Printf("Bắt đầu theo dõi giao dịch cho %s tại %s", token.Symbol, streamURL)

		go func(token TokenConfig) {
			for {
				conn, err := ConnectWebSocket(streamURL)
				if err != nil {
					mainLogger.Printf("Lỗi kết nối WebSocket cho %s: %v, thử lại sau 10 giây", token.Symbol, err)
					time.Sleep(10 * time.Second)
					continue
				}

				mainLogger.Printf("Đã kết nối thành công đến WebSocket cho %s", token.Symbol)
				go Heartbeat(conn, "market-"+token.Symbol)

				for {
					select {
					case <-stopChan:
						conn.Close()
						mainLogger.Printf("Đóng kết nối WebSocket cho %s", token.Symbol)
						return
					default:
						_, message, err := conn.ReadMessage()
						if err != nil {
							mainLogger.Printf("Lỗi đọc tin nhắn WebSocket cho %s: %v, kết nối lại", token.Symbol, err)
							conn.Close()
							break
						}

						var trade TradeEvent
						if err := json.Unmarshal(message, &trade); err != nil {
							mainLogger.Printf("Lỗi parse JSON cho %s: %v", token.Symbol, err)
							continue
						}

						asset := strings.TrimSuffix(trade.Symbol, "USDT")
						quantity := ParseFloat(trade.Quantity)
						price := ParseFloat(trade.Price)

						// Log tất cả giao dịch vào logger tương ứng
						logger := getLoggerForSymbol(trade.Symbol)
						orderType := "MUA"
						if trade.IsBuyer {
							orderType = "BÁN"
						}

						logger.Printf("Giao dịch: %s | Số lượng: %.8f | Giá: %.2f USDT | Tổng: %.2f USDT",
							orderType, quantity, price, quantity*price)

						// Chỉ xử lý giao dịch lớn
						if quantity > token.LargeOrderAmount {
							// Tạo Order mới
							newOrder := BTCOrder{
								Timestamp: time.UnixMilli(trade.Timestamp),
								OrderID:   fmt.Sprintf("%d-%s", trade.Timestamp, asset),
								Symbol:    asset,
								Side:      orderType,
								Amount:    quantity,
								Price:     price,
								Source:    "RealTime",
							}

							// Lưu vào biến global
							realTimeOrdersMutex.Lock()
							realTimeOrders = append(realTimeOrders, newOrder)

							// Log giao dịch lớn
							orderSize := "LỚN"
							if quantity >= token.HugeOrderAmount {
								orderSize = "RẤT LỚN"
							}

							mainLogger.Printf("[GIAO DỊCH %s] %s | %s | Số lượng: %.8f | Giá: %.2f USDT | Tổng: %.2f USDT",
								orderSize, asset, orderType, quantity, price, quantity*price)

							realTimeOrdersMutex.Unlock()
						}
					}
				}
			}
		}(token)
	}
}

func ProcessRealTimeKlineStream(cfg BinanceConfig, stopChan <-chan struct{}) {
	for _, token := range cfg.Tokens {
		// Sử dụng URL trực tiếp như trong file real_tIme_BTC.go
		streamURL := "wss://stream.binance.com:9443/ws/" + strings.ToLower(token.Symbol) + "@kline_1m"

		mainLogger.Printf("Bắt đầu theo dõi kline cho %s tại %s", token.Symbol, streamURL)

		go func(token TokenConfig) {
			for {
				conn, err := ConnectWebSocket(streamURL)
				if err != nil {
					mainLogger.Printf("Lỗi kết nối WebSocket kline cho %s: %v, thử lại sau 5 giây", token.Symbol, err)
					time.Sleep(5 * time.Second)
					continue
				}

				mainLogger.Printf("Đã kết nối thành công đến WebSocket kline cho %s", token.Symbol)
				go Heartbeat(conn, "kline-"+token.Symbol)

				for {
					select {
					case <-stopChan:
						conn.Close()
						mainLogger.Printf("Đóng kết nối WebSocket kline cho %s", token.Symbol)
						return
					default:
						_, message, err := conn.ReadMessage()
						if err != nil {
							mainLogger.Printf("Lỗi đọc tin nhắn WebSocket kline cho %s: %v, kết nối lại", token.Symbol, err)
							conn.Close()
							break
						}

						var kline KlineEvent
						if err := json.Unmarshal(message, &kline); err != nil {
							mainLogger.Printf("Lỗi parse JSON kline cho %s: %v", token.Symbol, err)
							continue
						}

						closePrice := ParseFloat(kline.Kline.Close)
						openPrice := ParseFloat(kline.Kline.Open)
						priceChange := (closePrice - openPrice) / openPrice * 100

						// Log thông tin kline vào logger tương ứng
						logger := getLoggerForSymbol(kline.Symbol)
						logger.Printf("Kline 1m: Mở: %.2f | Đóng: %.2f | Thay đổi: %.2f%%",
							openPrice, closePrice, priceChange)

						// Xử lý thay đổi giá đáng kể
						if math.Abs(priceChange) >= cfg.PriceChangeThreshold {
							asset := strings.TrimSuffix(kline.Symbol, "USDT")
							changeType := "TĂNG"
							if priceChange < 0 {
								changeType = "GIẢM"
							}

							mainLogger.Printf("[BIẾN ĐỘNG GIÁ] %s %s %.2f%% trong 1 phút (%.2f -> %.2f USDT)",
								asset, changeType, math.Abs(priceChange), openPrice, closePrice)

							logger.Printf("[CẢNH BÁO] Biến động giá lớn: %s %.2f%% trong 1 phút (%.2f -> %.2f USDT)",
								changeType, math.Abs(priceChange), openPrice, closePrice)
						}
					}
				}
			}
		}(token)
	}
}

// HandleRealTimeCrypto xử lý dữ liệu tiền điện tử thời gian thực từ Binance
func HandleRealTimeCrypto(configFile string, stopChan <-chan struct{}) {
	// Thiết lập logger
	if err := setupLoggers(); err != nil {
		fmt.Printf("Lỗi khi thiết lập logger: %v\n", err)
		return
	}

	mainLogger.Println("======= BẮT ĐẦU XỬ LÝ DỮ LIỆU TIỀN ĐIỆN TỬ THỜI GIAN THỰC =======")

	cfg, err := LoadBinanceConfig(configFile)
	if err != nil {
		mainLogger.Printf("Lỗi khi đọc file cấu hình: %v", err)
		return
	}

	mainLogger.Printf("Đã tải cấu hình từ file: %s", configFile)
	mainLogger.Printf("WebSocket URL: %s", cfg.WSURL)
	mainLogger.Printf("Số lượng token cần theo dõi: %d", len(cfg.Tokens))
	for i, token := range cfg.Tokens {
		mainLogger.Printf("Token #%d: %s (Giao dịch lớn: %f, Giao dịch rất lớn: %f)",
			i+1, token.Symbol, token.LargeOrderAmount, token.HugeOrderAmount)
	}

	go ProcessRealTimeMarketStream(cfg, stopChan)
	go ProcessRealTimeKlineStream(cfg, stopChan)

	<-stopChan
	mainLogger.Println("======= KẾT THÚC XỬ LÝ DỮ LIỆU TIỀN ĐIỆN TỬ THỜI GIAN THỰC =======")
}

// GetRealTimeOrders lấy dữ liệu giao dịch thời gian thực
func GetRealTimeOrders() []BTCOrder {
	realTimeOrdersMutex.RLock()
	defer realTimeOrdersMutex.RUnlock()

	// Tạo bản sao của dữ liệu để tránh race condition
	ordersCopy := make([]BTCOrder, len(realTimeOrders))
	copy(ordersCopy, realTimeOrders)

	return ordersCopy
}

// SummarizeOrderData tổng hợp dữ liệu giao dịch theo khoảng thời gian
func SummarizeOrderData(duration time.Duration) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for range ticker.C {
		realTimeOrdersMutex.RLock()
		orders := realTimeOrders
		realTimeOrdersMutex.RUnlock()

		if len(orders) == 0 {
			continue
		}

		// Tổng hợp dữ liệu theo loại tiền
		btcOrders := filterOrdersBySymbol(orders, "BTC")
		ethOrders := filterOrdersBySymbol(orders, "ETH")
		solOrders := filterOrdersBySymbol(orders, "SOL")

		// Ghi log tổng hợp
		mainLogger.Printf("===== TỔNG HỢP GIAO DỊCH LỚN TRONG %v QUA =====", duration)
		mainLogger.Printf("Tổng số giao dịch lớn: %d (BTC: %d, ETH: %d, SOL: %d)",
			len(orders), len(btcOrders), len(ethOrders), len(solOrders))

		// Ghi log chi tiết cho từng loại tiền
		logOrderSummary(btcLogger, btcOrders, "BTC")
		logOrderSummary(ethLogger, ethOrders, "ETH")
		logOrderSummary(solLogger, solOrders, "SOL")

		// Xóa dữ liệu đã tổng hợp
		realTimeOrdersMutex.Lock()
		realTimeOrders = []BTCOrder{}
		realTimeOrdersMutex.Unlock()
	}
}

// filterOrdersBySymbol lọc giao dịch theo loại tiền
func filterOrdersBySymbol(orders []BTCOrder, symbol string) []BTCOrder {
	var result []BTCOrder
	for _, order := range orders {
		if order.Symbol == symbol {
			result = append(result, order)
		}
	}
	return result
}

// logOrderSummary ghi log tổng hợp giao dịch
func logOrderSummary(logger *log.Logger, orders []BTCOrder, symbol string) {
	if len(orders) == 0 {
		return
	}

	var totalBuyAmount, totalSellAmount, totalBuyValue, totalSellValue float64
	var buyCount, sellCount int

	for _, order := range orders {
		if order.Side == "MUA" {
			totalBuyAmount += order.Amount
			totalBuyValue += order.Amount * order.Price
			buyCount++
		} else {
			totalSellAmount += order.Amount
			totalSellValue += order.Amount * order.Price
			sellCount++
		}
	}

	logger.Printf("===== TỔNG HỢP GIAO DỊCH %s =====", symbol)
	logger.Printf("Tổng số giao dịch: %d (Mua: %d, Bán: %d)", len(orders), buyCount, sellCount)
	logger.Printf("Khối lượng mua: %.8f %s (%.2f USDT)", totalBuyAmount, symbol, totalBuyValue)
	logger.Printf("Khối lượng bán: %.8f %s (%.2f USDT)", totalSellAmount, symbol, totalSellValue)

	netFlow := totalBuyAmount - totalSellAmount
	netFlowValue := totalBuyValue - totalSellValue
	flowDirection := "VÀO"
	if netFlow < 0 {
		flowDirection = "RA"
		netFlow = -netFlow
		netFlowValue = -netFlowValue
	}

	logger.Printf("Dòng tiền ròng: %.8f %s (%.2f USDT) %s thị trường",
		netFlow, symbol, netFlowValue, flowDirection)
}

// HandleCryptoData xử lý dữ liệu từ cả BTC, ETH và SOL
func HandleCryptoData(configFile string) {
	// Thiết lập logger
	if err := setupLoggers(); err != nil {
		fmt.Printf("Lỗi khi thiết lập logger: %v\n", err)
		return
	}

	mainLogger.Println("======= BẮT ĐẦU XỬ LÝ DỮ LIỆU TIỀN ĐIỆN TỬ =======")

	// Đọc cấu hình
	cfg, err := LoadBinanceConfig(configFile)
	if err != nil {
		mainLogger.Printf("Lỗi khi đọc file cấu hình: %v", err)
		return
	}

	mainLogger.Printf("Đã tải cấu hình từ file: %s", configFile)
	mainLogger.Printf("WebSocket URL: %s", cfg.WSURL)
	mainLogger.Printf("Số lượng token cần theo dõi: %d", len(cfg.Tokens))
	for i, token := range cfg.Tokens {
		mainLogger.Printf("Token #%d: %s (Giao dịch lớn: %f, Giao dịch rất lớn: %f)",
			i+1, token.Symbol, token.LargeOrderAmount, token.HugeOrderAmount)
	}

	// Kênh dừng để kiểm soát goroutines
	stopChan := make(chan struct{})

	// Khởi động xử lý dữ liệu thời gian thực
	go ProcessRealTimeMarketStream(cfg, stopChan)
	go ProcessRealTimeKlineStream(cfg, stopChan)

	// Khởi động tổng hợp dữ liệu định kỳ
	go SummarizeOrderData(5 * time.Minute)

	// Xử lý tín hiệu để dừng chương trình một cách an toàn
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	mainLogger.Println("Nhận tín hiệu dừng, đang kết thúc xử lý...")
	close(stopChan)
	time.Sleep(2 * time.Second) // Đợi các goroutine kết thúc
	mainLogger.Println("======= KẾT THÚC XỬ LÝ DỮ LIỆU TIỀN ĐIỆN TỬ =======")
}

// Hàm chính để chạy ứng dụng
func RunCryptoDataProcessor(configPath string) {
	fmt.Println("Bắt đầu xử lý dữ liệu tiền điện tử...")
	HandleCryptoData(configPath)
}
