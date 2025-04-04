package ohlcv

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var lastKlinesWeek = make(map[string]Kline) // Lưu Kline cuối cùng cho từng symbol

// FetchHistoricalData lấy dữ liệu lịch sử cho tất cả các coin
func historicalDataWeek() {
	for _, coin := range coins {
		url := fmt.Sprintf(BinanceAPIBaseURL, strings.ToUpper(coin), "1w")
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Error fetching historical data for %s: %v", coin, err)
			continue
		}
		defer resp.Body.Close()

		var klines [][]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&klines); err != nil {
			log.Printf("Error decoding JSON for %s: %v", coin, err)
			continue
		}

		fmt.Printf("Historical Data for %s:\n", strings.ToUpper(coin))
		for _, k := range klines {
			responseOHLCV := ResponseOHLCV{
				Symbol:           strings.ToUpper(coin),
				OpenTime:         new(big.Int).SetUint64(uint64(k[0].(float64)) + 271158000),
				Open:             strToBigInt(k[1].(string)),
				High:             strToBigInt(k[2].(string)),
				Low:              strToBigInt(k[3].(string)),
				Close:            strToBigInt(k[4].(string)),
				Volume:           k[5].(string),
				CloseTime:        new(big.Int).SetUint64(uint64(k[6].(float64)) + 271158000),
				QuoteAssetVolume: k[7].(string),
				NumberOfTrades:   new(big.Int).SetUint64(uint64(k[8].(float64))),
				TakerBuyBaseVol:  k[9].(string),
				TakerBuyQuoteVol: k[10].(string),
			}
			// ConnectToSMCWeek(responseOHLCV)
			fmt.Printf("Symbol: %s | OpenTime: %d | Open: %s | High: %s | Low: %s | Close: %s | Volume: %s | CloseTime: %d | QuoteVolume: %s | Trades: %d | TakerBuyBase: %s | TakerBuyQuote: %s\n",
				responseOHLCV.Symbol, responseOHLCV.OpenTime, responseOHLCV.Open, responseOHLCV.High, responseOHLCV.Low, responseOHLCV.Close, k[5].(string), responseOHLCV.CloseTime, k[7].(string), int(k[8].(float64)), k[9].(string), k[10].(string))
			timetemp[coin] = responseOHLCV.OpenTime
		}
	}
}

// ListenWebSocketOHLCV lắng nghe dữ liệu real-time cho tất cả các coin một cách tuần tự
func WebSocketOHLCVWeek() {
	historicalDataWeek()
	for _, coin := range coins {
		wsURL := fmt.Sprintf("%s/%s@kline_1w", BinanceWSBaseURL, coin)
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			log.Printf("Error connecting to WebSocket for %s: %v", coin, err)
			continue
		}
		defer conn.Close()

		fmt.Printf("Connected to Binance WebSocket for %s\n", coin)

		lastMessageTime := time.Now()

		// Kiểm tra timeout
		timeoutChan := make(chan bool)
		go func() {
			for {
				time.Sleep(10 * time.Second)
				if time.Since(lastMessageTime) > 1*time.Minute {
					fmt.Printf("Warning: No WebSocket data received for %s for over 1 minute!\n", coin)
					timeoutChan <- true
					return
				}
			}
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading message for %s: %v", coin, err)
				break
			}

			var wsData WSMessage
			if err := json.Unmarshal(message, &wsData); err != nil {
				log.Printf("Error unmarshalling message for %s: %v", coin, err)
				continue
			}

			lastMessageTime = time.Now()

			kline := wsData.Kline
			openTime := new(big.Int).SetUint64(uint64(kline.OpenTime) + 271158000)
			closeTime := kline.CloseTime

			if kline.OpenTime != lastKlinesWeek[coin].OpenTime && timetemp[coin] != openTime {
				// responseOHLCV := ResponseOHLCV{
				// 	Symbol:           kline.Symbol,
				// 	OpenTime:         openTime,
				// 	Open:             strToBigInt(string(kline.Open)),
				// 	High:             strToBigInt(string(kline.High)),
				// 	Low:              strToBigInt(string(kline.Low)),
				// 	Close:            strToBigInt(string(kline.Close)),
				// 	Volume:           string(kline.Volume),
				// 	CloseTime:        new(big.Int).SetUint64(uint64(kline.CloseTime) + 271158000),
				// 	QuoteAssetVolume: string(kline.QuoteAssetVolume),
				// 	NumberOfTrades:   new(big.Int).SetUint64(uint64(kline.NumberOfTrades)),
				// 	TakerBuyBaseVol:  string(kline.TakerBuyBaseVolume),
				// 	TakerBuyQuoteVol: string(kline.TakerBuyQuoteVolume),
				// }
				// ConnectToSMCWeek(responseOHLCV)
				fmt.Printf("Real-time Data - Symbol: %s | OpenTime: %d | Open: %s | High: %s | Low: %s | Close: %s | Volume: %s | CloseTime: %d | QuoteVolume: %s | Trades: %d | TakerBuyBase: %s | TakerBuyQuote: %s\n",
					kline.Symbol, openTime, kline.Open, kline.High, kline.Low, kline.Close, kline.Volume, closeTime, kline.QuoteAssetVolume, kline.NumberOfTrades, kline.TakerBuyBaseVolume, kline.TakerBuyQuoteVolume)
				lastKlinesWeek[coin] = kline
			}

			// Kiểm tra nếu có timeout thì thoát vòng lặp để chuyển sang coin tiếp theo
			select {
			case <-timeoutChan:
				fmt.Printf("Timeout detected for %s, moving to next coin\n", coin)
				return
			default:
				continue
			}
		}
	}
}
