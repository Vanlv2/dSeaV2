package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

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

type BinanceTransaction struct {
	EventType       string
	TransactionType string
	OrderType       string
	Asset           string
	Amount          float64
	Price           string
	Timestamp       time.Time
	Collection      string
	SignalName      string
	SignalContent   string
	ChainFilter     string
	AlertType       string
	AlertStatus     float64
	AlertID         string
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

// func saveBinanceTransaction(db *sql.DB, tx BinanceTransaction) error {
//     var query string
//     var args []interface{}

//     // Dựa vào trường Collection để xác định bảng cần chèn dữ liệu
//     switch tx.Collection {
//     case "large_orders":
//         query = `
//             INSERT INTO large_orders (timestamp, order_id, symbol, side, amount, price, created_at)
//             VALUES (?, ?, ?, ?, ?, ?, NOW())
//         `
//         orderId := fmt.Sprintf("%d-%s", tx.Timestamp.UnixNano(), tx.Asset)
//         args = []interface{}{tx.Timestamp, orderId, tx.Asset, tx.OrderType, tx.Amount, BinanceparseFloat(tx.Price)}

//     case "huge_orders":
//         query = `
//             INSERT INTO huge_orders (timestamp, order_id, symbol, side, amount, price, created_at)
//             VALUES (?, ?, ?, ?, ?, ?, NOW())
//         `
//         orderId := fmt.Sprintf("%d-%s", tx.Timestamp.UnixNano(), tx.Asset)
//         args = []interface{}{tx.Timestamp, orderId, tx.Asset, tx.OrderType, tx.Amount, BinanceparseFloat(tx.Price)}

//     case "price_changes":
//         // Lưu vào bảng price_changes
//         query = `
//             INSERT INTO price_changes (timestamp, asset, alert_type, price_change, current_price)
//             VALUES (?, ?, ?, ?, ?)
//         `
//         args = []interface{}{tx.Timestamp, tx.Asset, tx.AlertType, tx.Amount, BinanceparseFloat(tx.Price)}

//         _, err := db.Exec(query, args...)
//         if err != nil {
//             return fmt.Errorf("lỗi khi lưu vào price_changes: %v", err)
//         }

//         // Lưu vào bảng price_signals
//         query = `
//             INSERT INTO price_signals (timestamp, signal_name, symbol, price, description)
//             VALUES (?, ?, ?, ?, ?)
//         `
//         args = []interface{}{tx.Timestamp, tx.SignalName, tx.Asset, BinanceparseFloat(tx.Price), tx.SignalContent}

//         _, err = db.Exec(query, args...)
//         if err != nil {
//             return fmt.Errorf("lỗi khi lưu vào price_signals: %v", err)
//         }

//         // Lưu vào bảng price_alerts
//         query = `
//             INSERT INTO price_alerts (timestamp, symbol, alert_price, current_price, alert_type)
//             VALUES (?, ?, ?, ?, ?)
//         `
//         args = []interface{}{tx.Timestamp, tx.Asset, tx.AlertStatus, BinanceparseFloat(tx.Price), tx.AlertType}

//         _, err = db.Exec(query, args...)
//         if err != nil {
//             return fmt.Errorf("lỗi khi lưu vào price_alerts: %v", err)
//         }

//         // Cập nhật market_analytics (giả định các thông tin phù hợp)
//         // Chúng ta có thể cần một API riêng để lấy thông tin này
//         query = `
//             INSERT INTO market_analytics (timestamp, market_cap, btc_dominance, volume_24h, fear_greed_index)
//             VALUES (?, ?, ?, ?, ?)
//         `
//         // Giả định một số giá trị hoặc gọi API để lấy dữ liệu chính xác
//         marketCap := 0.0       // Cần API để lấy dữ liệu này
//         btcDominance := 0.0    // Cần API để lấy dữ liệu này
//         volume24h := 0.0       // Có thể ước tính từ dữ liệu khối lượng hiện tại
//         fearGreedIndex := 0    // Cần API để lấy dữ liệu này

//         args = []interface{}{tx.Timestamp, marketCap, btcDominance, volume24h, fearGreedIndex}

//         // Ở đây, chúng ta có thể quyết định xem có nên thực sự lưu dữ liệu này không
//         // vì chúng ta có thể không có đủ thông tin
//         // _, err = db.Exec(query, args...)
//         // if err != nil {
//         //     return fmt.Errorf("lỗi khi lưu vào market_analytics: %v", err)
//         // }

//         return nil

//     case "buy_sell_btc_with_large_amount":
//         query = `
//             INSERT INTO buy_sell_btc_with_large_amount (timestamp, order_id, side, amount, price, created_at)
//             VALUES (?, ?, ?, ?, ?, NOW())
//         `
//         orderId := fmt.Sprintf("%d-%s", tx.Timestamp.UnixNano(), tx.Asset)
//         args = []interface{}{tx.Timestamp, orderId, tx.OrderType, tx.Amount, BinanceparseFloat(tx.Price)}

//     default:
//         return fmt.Errorf("không tìm thấy bảng phù hợp cho collection: %s", tx.Collection)
//     }

//     // Thực thi truy vấn cho các trường hợp không phải là price_changes
//     // vì price_changes đã được xử lý đặc biệt ở trên
//     if tx.Collection != "price_changes" {
//         _, err := db.Exec(query, args...)
//         return err
//     }

//     return nil
// }

func SaveBinanceTransaction(db *sql.DB, tx BinanceTransaction) error {
	var query string
	var args []interface{}

	// Dựa vào trường Collection để xác định bảng cần chèn dữ liệu
	switch tx.Collection {
	case "large_orders":
		query = `
            INSERT INTO large_orders (timestamp, order_id, symbol, side, amount, price, created_at)
            VALUES (?, ?, ?, ?, ?, ?, NOW())
        `
		orderId := fmt.Sprintf("%d-%s", tx.Timestamp.UnixNano(), tx.Asset)
		args = []interface{}{tx.Timestamp, orderId, tx.Asset, tx.OrderType, tx.Amount, BinanceparseFloat(tx.Price)}

	case "huge_orders":
		query = `
            INSERT INTO huge_orders (timestamp, order_id, symbol, side, amount, price, created_at)
            VALUES (?, ?, ?, ?, ?, ?, NOW())
        `
		orderId := fmt.Sprintf("%d-%s", tx.Timestamp.UnixNano(), tx.Asset)
		args = []interface{}{tx.Timestamp, orderId, tx.Asset, tx.OrderType, tx.Amount, BinanceparseFloat(tx.Price)}

	case "price_changes":
		// Lưu vào bảng price_changes
		query = `
            INSERT INTO price_changes (timestamp, asset, alert_type, price_change, current_price)
            VALUES (?, ?, ?, ?, ?)
        `
		args = []interface{}{tx.Timestamp, tx.Asset, tx.AlertType, tx.Amount, BinanceparseFloat(tx.Price)}

		_, err := db.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("lỗi khi lưu vào price_changes: %v", err)
		}

		// Lưu vào bảng price_signals
		query = `
            INSERT INTO price_signals (timestamp, signal_name, symbol, price, description)
            VALUES (?, ?, ?, ?, ?)
        `
		args = []interface{}{tx.Timestamp, tx.SignalName, tx.Asset, BinanceparseFloat(tx.Price), tx.SignalContent}

		_, err = db.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("lỗi khi lưu vào price_signals: %v", err)
		}

		// Lưu vào bảng price_alerts
		query = `
            INSERT INTO price_alerts (timestamp, symbol, alert_price, current_price, alert_type)
            VALUES (?, ?, ?, ?, ?)
        `
		args = []interface{}{tx.Timestamp, tx.Asset, tx.AlertStatus, BinanceparseFloat(tx.Price), tx.AlertType}

		_, err = db.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("lỗi khi lưu vào price_alerts: %v", err)
		}

		return nil

	case "buy_sell_btc_with_large_amount":
		query = `
            INSERT INTO buy_sell_btc_with_large_amount (timestamp, order_id, side, amount, price, created_at)
            VALUES (?, ?, ?, ?, ?, NOW())
        `
		orderId := fmt.Sprintf("%d-%s", tx.Timestamp.UnixNano(), tx.Asset)
		args = []interface{}{tx.Timestamp, orderId, tx.OrderType, tx.Amount, BinanceparseFloat(tx.Price)}

	default:
		return fmt.Errorf("không tìm thấy bảng phù hợp cho collection: %s", tx.Collection)
	}

	// Thực thi truy vấn cho các trường hợp không phải là price_changes
	// vì price_changes đã được xử lý đặc biệt ở trên
	if tx.Collection != "price_changes" {
		_, err := db.Exec(query, args...)
		return err
	}

	return nil
}

func connect_database() (*sql.DB, error) {
	connectionString := "root:@ztegc4df9f4e@tcp(172.21.0.4:3306)/binance"

	log.Printf("Đang kết nối đến database với connection string: %s",
		strings.Replace(connectionString, "root:", "root:[PASSWORD_HIDDEN]", 1))

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi mở kết nối database: %v", err)
	}

	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("không thể ping đến database: %v", err)
	}

	log.Printf("✅ Kết nối đến database thành công")
	return db, nil
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

func ConnectWebSocket(url string, logger *log.Logger) (*websocket.Conn, error) {
	logger.Printf("[DEBUG] Connecting to WebSocket URL: %s", url)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
func Heartbeat(conn *websocket.Conn, name string, logger *log.Logger) {
	for {
		time.Sleep(10 * time.Second)
		if err := conn.WriteMessage(websocket.PingMessage, []byte("ping-"+name)); err != nil {
			logger.Printf("[Binance] Heartbeat failed for %s: %v", name, err)
			return
		}
	}
}

func BinanceparseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func ProcessMarketStream(cfg BinanceConfig, db *sql.DB, logger *log.Logger, stopChan <-chan struct{}) {
	for _, token := range cfg.Tokens {
		streamURL := "wss://stream.binance.com:9443/ws/" + strings.ToLower(token.Symbol) + "@trade"
		logger.Printf("[DEBUG] Market stream URL for %s: %s", token.Symbol, streamURL)
		go func(token TokenConfig) {
			for {
				conn, err := ConnectWebSocket(streamURL, logger)
				if err != nil {
					logger.Printf("[Binance] Market WebSocket connection failed for %s: %v", token.Symbol, err)
					time.Sleep(5 * time.Second)
					continue
				}
				logger.Printf("[Binance] Connected to market WebSocket for %s trades...", token.Symbol)
				go Heartbeat(conn, "market-"+token.Symbol, logger)

				for {
					select {
					case <-stopChan:
						conn.Close()
						return
					default:
						_, message, err := conn.ReadMessage()
						if err != nil {
							logger.Printf("[Binance] Market WebSocket read error for %s: %v", token.Symbol, err)
							conn.Close()
							break
						}
						logger.Printf("[Binance] Received raw message for %s: %s", token.Symbol, string(message))
						var trade TradeEvent
						if err := json.Unmarshal(message, &trade); err != nil {
							logger.Printf("[Binance] Market JSON unmarshal error for %s: %v", token.Symbol, err)
							continue
						}
						logger.Printf("[Binance] Raw trade: Symbol=%s, Quantity=%s, IsBuyer=%v", trade.Symbol, trade.Quantity, trade.IsBuyer)

						asset := strings.TrimSuffix(trade.Symbol, "USDT")
						quantity := BinanceparseFloat(trade.Quantity)
						if quantity > token.LargeOrderAmount {
							orderType := "buy"
							if trade.IsBuyer {
								orderType = "sell"
							}
							collection := "large_orders"
							if quantity > token.HugeOrderAmount {
								collection = "huge_orders"
							}
							transaction := BinanceTransaction{
								EventType:       "trade",
								TransactionType: "market_trade",
								OrderType:       orderType,
								Asset:           asset,
								Amount:          quantity,
								Price:           trade.Price,
								Timestamp:       time.UnixMilli(trade.Timestamp),
								Collection:      collection,
							}
							if err := SaveBinanceTransaction(db, transaction); err != nil {
								logger.Printf("[Binance] Failed to save transaction: %v", err)
							} else {
								logger.Printf("[Binance] Recorded %s %s order: Amount=%f, Price=%s in %s", asset, orderType, quantity, trade.Price, collection)
							}
						}
					}
				}
			}
		}(token)
	}
}

func ProcessKlineStream(cfg BinanceConfig, txChan chan<- interface{}, logger *log.Logger, stopChan <-chan struct{}) {
	for _, token := range cfg.Tokens {
		streamURL := "wss://stream.binance.com:9443/ws/" + strings.ToLower(token.Symbol) + "@kline_1m"
		logger.Printf("[DEBUG] Kline stream URL for %s: %s", token.Symbol, streamURL)
		go func(token TokenConfig) {
			for {
				conn, err := ConnectWebSocket(streamURL, logger)
				if err != nil {
					logger.Printf("[Binance] Kline WebSocket connection failed for %s: %v", token.Symbol, err)
					time.Sleep(5 * time.Second)
					continue
				}
				logger.Printf("[Binance] Connected to kline WebSocket for %s...", token.Symbol)
				go Heartbeat(conn, "kline-"+token.Symbol, logger)

				for {
					select {
					case <-stopChan:
						conn.Close()
						return
					default:
						_, message, err := conn.ReadMessage()
						if err != nil {
							logger.Printf("[Binance] Kline WebSocket read error for %s: %v", token.Symbol, err)
							conn.Close()
							break
						}
						logger.Printf("[Binance] Received raw kline message for %s: %s", token.Symbol, string(message))
						var kline KlineEvent
						if err := json.Unmarshal(message, &kline); err != nil {
							logger.Printf("[Binance] Kline JSON unmarshal error for %s: %v", token.Symbol, err)
							continue
						}

						// if !kline.Kline.IsClosed {
						// 	continue
						// } không biết là điều kiện gì nhưng mà thôi lỗi do này nên để đó

						closePrice := BinanceparseFloat(kline.Kline.Close)
						openPrice := BinanceparseFloat(kline.Kline.Open)
						priceChange := (closePrice - openPrice) / openPrice * 100
						logger.Println("priceChange là:", priceChange)
						fmt.Printf("-cfg.PriceChangeThreshold: %v\n", -cfg.PriceChangeThreshold)
						logger.Printf("[Binance] Kline raw: Symbol=%s, Open=%f, Close=%f, Change=%.4f%%", kline.Symbol, openPrice, closePrice, priceChange)

						if priceChange > cfg.PriceChangeThreshold || priceChange < -cfg.PriceChangeThreshold {
							asset := strings.TrimSuffix(kline.Symbol, "USDT")
							var signalContent, alertType string
							if priceChange > 0 {
								signalContent = fmt.Sprintf("%s đang biến động đẩy mạnh", asset)
								alertType = "mua vào"
							} else {
								signalContent = fmt.Sprintf("%s đang biến động giảm mạnh", asset)
								alertType = "bán gấp"
							}
							fake15mChange := priceChange * 1.5
							fake1hChange := priceChange * 3.0
							alertID := fmt.Sprintf("15m:%.4f%%,1h:%.4f%%", fake15mChange, fake1hChange)

							transaction := BinanceTransaction{
								EventType:       "kline",
								TransactionType: "price_change",
								OrderType:       "",
								Asset:           asset,
								Amount:          priceChange,
								Price:           kline.Kline.Close,
								Timestamp:       time.UnixMilli(kline.Kline.StartTime),
								Collection:      "price_changes",
								SignalName:      asset,
								SignalContent:   signalContent,
								ChainFilter:     "Binance Chain",
								AlertType:       alertType,
								AlertStatus:     priceChange,
								AlertID:         alertID,
							}
							txChan <- transaction
							logger.Printf("[Binance] Price change detected for %s: %.4f%% (Open: %.2f, Close: %.2f) - Signal: %s, Alert: %s", asset, priceChange, openPrice, closePrice, signalContent, alertType)
						}
					}
				}
			}
		}(token)
	}
}

func HandleBinance(configFile string, txChan chan<- interface{}, stopChan <-chan struct{}, logger *log.Logger) {
	db, err := connect_database()
	if err != nil {
		logger.Printf("[Binance] Failed to connect to database: %v", err)
		return
	}
	defer db.Close()

	cfg, err := LoadBinanceConfig(configFile)
	if err != nil {
		logger.Printf("[Binance] Failed to load config: %v", err)
		return
	}
	go ProcessMarketStream(cfg, db, logger, stopChan)
	go ProcessKlineStream(cfg, txChan, logger, stopChan)
	<-stopChan
}
