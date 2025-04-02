package narrativesPerforments

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// BinanceTickerResponse là cấu trúc dữ liệu từ Binance WebSocket
type BinanceTickerResponse struct {
	EventType        string          `json:"e"` // Loại sự kiện (chuỗi "24hrTicker")
	EventTime        int64           `json:"E"` // Thời gian sự kiện (timestamp)
	Symbol           string          `json:"s"` // Symbol, ví dụ: "BTCUSDT"
	PriceChange      json.RawMessage `json:"p"` // Thay đổi giá 24h (price change)
	Change24h        json.RawMessage `json:"P"` // Thay đổi giá 24h (%) (percent change)
	WeightedAvgPrice json.RawMessage `json:"w"` // Giá trung bình có trọng số 24h
	PrevClosePrice   json.RawMessage `json:"x"` // Giá đóng ngày trước
	CurrentPrice     json.RawMessage `json:"c"` // Giá hiện tại
	LastQuantity     json.RawMessage `json:"Q"` // Khối lượng giao dịch cuối cùng
	BestBidPrice     json.RawMessage `json:"b"` // Giá bid tốt nhất
	BestBidQuantity  json.RawMessage `json:"B"` // Khối lượng bid tốt nhất
	BestAskPrice     json.RawMessage `json:"a"` // Giá ask tốt nhất
	BestAskQuantity  json.RawMessage `json:"A"` // Khối lượng ask tốt nhất
	OpenPrice        json.RawMessage `json:"o"` // Giá mở 24h
	HighPrice        json.RawMessage `json:"h"` // Giá cao nhất 24h
	LowPrice         json.RawMessage `json:"l"` // Giá thấp nhất 24h
	Volume           json.RawMessage `json:"v"` // Tổng khối lượng giao dịch 24h
	QuoteVolume      json.RawMessage `json:"q"` // Tổng giá trị giao dịch 24h
	OpenTime         int64           `json:"O"` // Thời gian bắt đầu 24h
	CloseTime        int64           `json:"C"` // Thời gian kết thúc 24h
	FirstTradeID     int64           `json:"F"` // ID giao dịch đầu tiên
	LastTradeID      int64           `json:"L"` // ID giao dịch cuối cùng
	NumberOfTrades   int64           `json:"n"` // Số lượng giao dịch 24h
}

// CoinGeckoResponse là cấu trúc dữ liệu từ CoinGecko API
type CoinGeckoResponse struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"` // Thêm trường name
	CirculatingSupply float64 `json:"circulating_supply"`
}

// Danh sách coin cần theo dõi
var coins = []string{
	"btcusdt", "ethusdt", "adausdt", "tonusdt", "nearusdt",
	"xlmusdt", "algousdt", "xtzusdt", "egldusdt", "ltcusdt",
	"xrpusdt", "bnbusdt", "avaxusdt", "trxusdt", "arbusdt",
	"opusdt", "atomusdt", "vetusdt", "solusdt",
	"dogeusdt", "shibusdt",
}

// Ánh xạ từ symbol Binance sang ID CoinGecko
var symbolToCoinGeckoID = map[string]string{
	"btcusdt":  "bitcoin",
	"ethusdt":  "ethereum",
	"adausdt":  "cardano",
	"tonusdt":  "the-open-network",
	"nearusdt": "near",
	"xlmusdt":  "stellar",
	"algousdt": "algorand",
	"xtzusdt":  "tezos",
	"egldusdt": "elrond-erd-2",
	"ltcusdt":  "litecoin",
	"xrpusdt":  "ripple",
	"bnbusdt":  "binancecoin",
	"avaxusdt": "avalanche-2",
	"trxusdt":  "tron",
	"arbusdt":  "arbitrum",
	"opusdt":   "optimism",
	"atomusdt": "cosmos",
	"vetusdt":  "vechain",
	"solusdt":  "solana",
	"dogeusdt": "dogecoin",
	"shibusdt": "shiba-inu",
}

// Cache để lưu trữ circulating supply và name
var coinCache = make(map[string]struct {
	Supply float64
	Name   string
})
var cacheLastUpdated time.Time

// convertToFloat64 chuyển đổi giá trị json.RawMessage thành float64
func convertToFloat64(raw json.RawMessage) (float64, error) {
	var f float64
	err := json.Unmarshal(raw, &f)
	if err == nil {
		return f, nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, fmt.Errorf("failed to unmarshal raw value %s: %v", string(raw), err)
	}
	_, err = fmt.Sscanf(s, "%f", &f)
	if err != nil {
		return 0, fmt.Errorf("failed to convert string %s to float64: %v", s, err)
	}
	return f, nil
}

// fetchAllCirculatingSupplies lấy dữ liệu circulating supply và name cho tất cả coin một lần
func fetchAllCirculatingSupplies() error {
	coinIDs := make([]string, 0, len(symbolToCoinGeckoID))
	for _, id := range symbolToCoinGeckoID {
		coinIDs = append(coinIDs, id)
	}
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&ids=%s", strings.Join(coinIDs, ","))
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch circulating supplies: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return fmt.Errorf("rate limit exceeded, status code: 429")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var data []CoinGeckoResponse
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return fmt.Errorf("failed to decode CoinGecko response: %v", err)
	}

	// Lưu trữ vào cache
	for _, item := range data {
		coinCache[item.ID] = struct {
			Supply float64
			Name   string
		}{
			Supply: item.CirculatingSupply,
			Name:   item.Name,
		}
	}
	cacheLastUpdated = time.Now()
	return nil
}

// getCirculatingSupply lấy circulating supply từ cache
func getCirculatingSupply(coinID string) (float64, error) {
	if time.Since(cacheLastUpdated) > 5*time.Minute {
		if err := fetchAllCirculatingSupplies(); err != nil {
			return 0, err
		}
	}

	data, exists := coinCache[coinID]
	if !exists {
		return 0, fmt.Errorf("no circulating supply data for coin ID: %s", coinID)
	}
	return data.Supply, nil
}

// getCoinName lấy tên coin từ cache
func getCoinName(coinID string) (string, error) {
	if time.Since(cacheLastUpdated) > 5*time.Minute {
		if err := fetchAllCirculatingSupplies(); err != nil {
			return "", err
		}
	}

	data, exists := coinCache[coinID]
	if !exists {
		return "", fmt.Errorf("no name data for coin ID: %s", coinID)
	}
	return data.Name, nil
}

// calculateMarketCap tính vốn hóa thị trường
func calculateMarketCap(currentPrice float64, coinID string) (float64, error) {
	supply, err := getCirculatingSupply(coinID)
	if err != nil {
		return 0, err
	}
	return currentPrice * supply, nil
}

// connectToBinanceWebSocket kết nối tới Binance WebSocket và log tất cả trường
func NarrativesPerforment() {
	// Khởi tạo cache ban đầu
	if err := fetchAllCirculatingSupplies(); err != nil {
		log.Printf("Failed to initialize circulating supply cache: %v", err)
		return
	}
	url := "wss://stream.binance.com:9443/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatalf("Failed to connect to Binance WebSocket: %v", err)
	}
	defer conn.Close()

	// Subscribe vào các kênh ticker
	subscribeMsg := map[string]interface{}{
		"method": "SUBSCRIBE",
		"params": func() []string {
			var params []string
			for _, coin := range coins {
				params = append(params, coin+"@ticker")
			}
			return params
		}(),
		"id": 1,
	}
	if err := conn.WriteJSON(subscribeMsg); err != nil {
		log.Fatalf("Failed to subscribe to ticker: %v", err)
	}

	// Đọc dữ liệu từ WebSocket
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			time.Sleep(5 * time.Second)
			NarrativesPerforment() // Thử kết nối lại
			return
		}

		// Parse dữ liệu vào struct
		var ticker BinanceTickerResponse
		if err := json.Unmarshal(message, &ticker); err != nil {
			log.Printf("Failed to parse ticker data: %v", err)
			continue
		}

		// Kiểm tra xem có phải là thông điệp ticker không
		if ticker.EventType != "24hrTicker" {
			continue
		}
		FullRecord(ticker) // Ghi dữ liệu vào file
	}
}

func FullRecord(ticker BinanceTickerResponse) {
	// Khởi động các hàm ghi dữ liệu
	RecordDay(ticker)
	RecordWeek(ticker)
	RecordMonth(ticker)
}
