package narrativesPerforments

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

func RecordDay(ticker BinanceTickerResponse) {
	// Chuyển đổi các giá trị json.RawMessage thành string
	var change24hStr, currentPriceStr, volumeStr, openPriceStr, highPriceStr, lowPriceStr, priceChange string
	if err := json.Unmarshal(ticker.Change24h, &change24hStr); err != nil {
		log.Printf("Failed to unmarshal Change24h: %v", err)
		return
	}
	if err := json.Unmarshal(ticker.CurrentPrice, &currentPriceStr); err != nil {
		log.Printf("Failed to unmarshal CurrentPrice: %v", err)
		return
	}
	if err := json.Unmarshal(ticker.Volume, &volumeStr); err != nil {
		log.Printf("Failed to unmarshal Volume: %v", err)
		return
	}
	if err := json.Unmarshal(ticker.OpenPrice, &openPriceStr); err != nil {
		log.Printf("Failed to unmarshal OpenPrice: %v", err)
		return
	}
	if err := json.Unmarshal(ticker.HighPrice, &highPriceStr); err != nil {
		log.Printf("Failed to unmarshal HighPrice: %v", err)
		return
	}
	if err := json.Unmarshal(ticker.LowPrice, &lowPriceStr); err != nil {
		log.Printf("Failed to unmarshal lowPriceStr: %v", err)
		return
	}
	if err := json.Unmarshal(ticker.PriceChange, &priceChange); err != nil {
		log.Printf("Failed to unmarshal PriceChange: %v", err)
		return
	}

	// Sử dụng trực tiếp ticker.Symbol vì nó đã là string
	symbolStr := ticker.Symbol

	// Lấy ID CoinGecko từ symbol Binance
	coinID := symbolToCoinGeckoID[strings.ToLower(symbolStr)]
	if coinID == "" {
		log.Printf("No CoinGecko ID mapping for %s", symbolStr)
		return
	}

	// Lấy tên coin từ CoinGecko
	coinName, err := getCoinName(coinID)
	if err != nil {
		log.Printf("Failed to get coin name for %s: %v", symbolStr, err)
		coinName = strings.ToUpper(strings.ReplaceAll(symbolStr, "USDT", "")) // Fallback
	}

	// Tính vốn hóa thị trường (cần currentPrice dưới dạng float64)
	currentPriceFloat, err := convertToFloat64(ticker.CurrentPrice)
	if err != nil {
		log.Printf("Failed to convert currentPrice to float64 for market cap: %v", err)
		return
	}
	marketCapFloat, err := calculateMarketCap(currentPriceFloat, coinID)
	if err != nil {
		log.Printf("Failed to calculate market cap for %s: %v", symbolStr, err)
		return
	}
	marketCapStr := fmt.Sprintf("%.2f", marketCapFloat) // Chuyển marketCap sang string

	// Truyền tất cả dưới dạng string vào ConnectToSmartContract
	ConnectToSMCDate(
		ticker.EventTime,
		coinName,
		openPriceStr,
		highPriceStr,
		lowPriceStr,
		currentPriceStr,
		volumeStr,
		"0",
		priceChange,
		change24hStr,
		marketCapStr,
		symbolStr,
	)
}
