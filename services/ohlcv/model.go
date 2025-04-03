package ohlcv

import (
	"encoding/json"
	"fmt"
	"math/big"
)

const (
	BinanceWSBaseURL  = "wss://stream.binance.com:9443/ws"
	BinanceAPIBaseURL = "https://api.binance.com/api/v3/klines?symbol=%s&interval=%s&limit=20"
)

// Danh sách các coin

var coins = []string{
	"btcusdt", "ethusdt", "adausdt", "tonusdt", "nearusdt",
	"xlmusdt", "algousdt", "xtzusdt", "egldusdt", "ltcusdt",
	"xrpusdt", "bnbusdt", "avaxusdt", "trxusdt", "arbusdt",
	"opusdt", "atomusdt", "vetusdt", "solusdt",
	"dogeusdt", "shibusdt",
}

// StringOrFloat là kiểu tùy chỉnh để xử lý cả string và float64
type StringOrFloat string

var timetemp = make(map[string]*big.Int) // Lưu thời gian cuối cùng cho từng symbol

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
	Open             string
	High             string
	Low              string
	Close            string
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
