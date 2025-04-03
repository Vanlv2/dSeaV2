package ohlcv

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strconv"
)

const (
	BinanceWSBaseURL  = "wss://stream.binance.com:9443/ws"
	BinanceAPIBaseURL = "https://api.binance.com/api/v3/klines?symbol=%s&interval=%s&limit=500"
)

// Danh sách các coin

var coins = []string{
	"btcusdt", "ethusdt", "adausdt", "tonusdt", "nearusdt",
	"xlmusdt", "algousdt", "xtzusdt", "egldusdt", "ltcusdt",
	"xrpusdt", "bnbusdt", "avaxusdt", "trxusdt", "arbusdt",
	"opusdt", "atomusdt", "vetusdt", "solusdt",
	"dogeusdt", "shibusdt",
}

// var coins = []string{
// 	"btcusdt",
// }

// StringOrFloat là kiểu tùy chỉnh để xử lý cả string và float64
type StringOrFloat string

var timetemp = make(map[string]*big.Int) // Lưu thời gian cuối cùng cho từng symbol

// Kline struct để lưu dữ liệu nến
type Kline struct {
	Symbol              string        `json:"s"`
	OpenTime            int64         `json:"t"`
	Open                StringOrFloat `json:"o"`
	High                StringOrFloat `json:"h"`
	Low                 StringOrFloat `json:"l"`
	Close               StringOrFloat `json:"c"`
	Volume              StringOrFloat `json:"v"`
	CloseTime           int64         `json:"T"`
	QuoteAssetVolume    StringOrFloat `json:"q"`
	NumberOfTrades      int           `json:"n"`
	TakerBuyBaseVolume  StringOrFloat `json:"V"`
	TakerBuyQuoteVolume StringOrFloat `json:"Q"`
}

type ResponseOHLCV struct {
	Symbol           string
	OpenTime         *big.Int
	Open             *big.Int
	High             *big.Int
	Low              *big.Int
	Close            *big.Int
	Volume           string
	CloseTime        *big.Int
	QuoteAssetVolume string
	NumberOfTrades   *big.Int
	TakerBuyBaseVol  string
	TakerBuyQuoteVol string
}

// WSMessage struct để ánh xạ dữ liệu từ WebSocket
type WSMessage struct {
	Event string `json:"e"`
	Time  int64  `json:"E"`
	Kline Kline  `json:"k"`
}

func strToBigInt(s string) *big.Int {
	// Chuyển đổi string thành float trước
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Printf("Lỗi: Không thể chuyển đổi string '%s' thành float: %v", s, err)
		return big.NewInt(0)
	}

	// Nhân với 10^8 để tránh mất dữ liệu phần thập phân
	scaledValue := new(big.Float).Mul(big.NewFloat(f), big.NewFloat(1e2))

	// Chuyển big.Float thành big.Int
	intValue := new(big.Int)
	scaledValue.Int(intValue) // Lấy phần nguyên

	return intValue
}
func (sf *StringOrFloat) UnmarshalJSON(b []byte) error {
	var val interface{}
	if err := json.Unmarshal(b, &val); err != nil {
		return err
	}
	switch v := val.(type) {
	case float64:
		*sf = StringOrFloat(fmt.Sprintf("%.8f", v))
	case string:
		*sf = StringOrFloat(v)
	default:
		return fmt.Errorf("invalid type for StringOrFloat: %T", val)
	}
	return nil
}
