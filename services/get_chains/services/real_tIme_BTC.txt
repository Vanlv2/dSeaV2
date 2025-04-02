package services

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
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

// Biến global để lưu trữ dữ liệu thời gian thực BTC
var realTimeBTCOrders []BTCOrder
var realTimeBTCOrdersMutex sync.RWMutex

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

func ProcessRealTimeMarketStream(cfg BinanceConfig, stopChan <-chan struct{}) {
	for _, token := range cfg.Tokens {
		streamURL := "wss://stream.binance.com:9443/ws/" + strings.ToLower(token.Symbol) + "@trade"

		go func(token TokenConfig) {
			for {
				conn, err := ConnectWebSocket(streamURL)
				if err != nil {
					time.Sleep(10 * time.Second)
					continue
				}

				go Heartbeat(conn, "market-"+token.Symbol)

				for {
					select {
					case <-stopChan:
						conn.Close()
						return
					default:
						_, message, err := conn.ReadMessage()
						if err != nil {
							conn.Close()
							break
						}

						var trade TradeEvent
						if err := json.Unmarshal(message, &trade); err != nil {
							continue
						}

						asset := strings.TrimSuffix(trade.Symbol, "USDT")
						quantity := ParseFloat(trade.Quantity)

						if quantity > token.LargeOrderAmount {
							orderType := "buy"
							if trade.IsBuyer {
								orderType = "sell"
							}

							// Tạo Order mới
							newOrder := BTCOrder{
								Timestamp: time.UnixMilli(trade.Timestamp),
								OrderID:   fmt.Sprintf("%d-%s", trade.Timestamp, asset),
								Symbol:    asset,
								Side:      orderType,
								Amount:    quantity,
								Price:     ParseFloat(trade.Price),
								Source:    "RealTime",
							}

							// Lưu vào biến global
							realTimeBTCOrdersMutex.Lock()
							realTimeBTCOrders = append(realTimeBTCOrders, newOrder)
							realTimeBTCOrdersMutex.Unlock()
						}
					}
				}
			}
		}(token)
	}
}

func ProcessRealTimeKlineStream(cfg BinanceConfig, stopChan <-chan struct{}) {
	for _, token := range cfg.Tokens {
		streamURL := "wss://stream.binance.com:9443/ws/" + strings.ToLower(token.Symbol) + "@kline_1m"

		go func(token TokenConfig) {
			for {
				conn, err := ConnectWebSocket(streamURL)
				if err != nil {
					time.Sleep(5 * time.Second)
					continue
				}

				go Heartbeat(conn, "kline-"+token.Symbol)

				for {
					select {
					case <-stopChan:
						conn.Close()
						return
					default:
						_, message, err := conn.ReadMessage()
						if err != nil {
							conn.Close()
							break
						}

						var kline KlineEvent
						if err := json.Unmarshal(message, &kline); err != nil {
							continue
						}

						closePrice := ParseFloat(kline.Kline.Close)
						openPrice := ParseFloat(kline.Kline.Open)
						priceChange := (closePrice - openPrice) / openPrice * 100

						// Xử lý thay đổi giá đáng kể
						if math.Abs(priceChange) >= cfg.PriceChangeThreshold {
							// Có thể thêm xử lý ở đây nếu cần
						}
					}
				}
			}
		}(token)
	}
}

// HandleRealTimeBTC xử lý dữ liệu BTC thời gian thực từ Binance
func HandleRealTimeBTC(configFile string, stopChan <-chan struct{}) {
	cfg, err := LoadBinanceConfig(configFile)
	if err != nil {
		return
	}

	go ProcessRealTimeMarketStream(cfg, stopChan)
	go ProcessRealTimeKlineStream(cfg, stopChan)

	<-stopChan
}

// GetRealTimeBTCOrders lấy dữ liệu giao dịch BTC thời gian thực
func GetRealTimeBTCOrders() []BTCOrder {
	realTimeBTCOrdersMutex.RLock()
	defer realTimeBTCOrdersMutex.RUnlock()

	// Tạo bản sao của dữ liệu để tránh race condition
	ordersCopy := make([]BTCOrder, len(realTimeBTCOrders))
	copy(ordersCopy, realTimeBTCOrders)

	return ordersCopy
}
