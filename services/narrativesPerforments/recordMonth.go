package narrativesPerforments

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type MonthlyData struct {
	OpenPrice  float64   // Giá mở cửa đầu tháng
	HighPrice  float64   // Giá cao nhất trong tháng
	LowPrice   float64   // Giá thấp nhất trong tháng
	ClosePrice float64   // Giá đóng cửa hiện tại (cập nhật liên tục)
	StartTime  time.Time // Thời gian bắt đầu tháng
	Volume     float64   // Tổng khối lượng giao dịch trong tháng
}

var monthlyData = make(map[string]*MonthlyData) // Thay weeklyData bằng monthlyData

func getMonthStart(t time.Time) time.Time {
	// Chuyển về UTC và đặt về ngày 1 của tháng
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func initializeMonthlyData(currentPrice float64, eventTime int64) *MonthlyData {
	eventTimeUTC := time.UnixMilli(eventTime).UTC()
	monthStart := getMonthStart(eventTimeUTC)
	return &MonthlyData{
		OpenPrice:  currentPrice,
		HighPrice:  currentPrice,
		LowPrice:   currentPrice,
		ClosePrice: currentPrice,
		StartTime:  monthStart,
		Volume:     0,
	}
}

func RecordMonth(ticker BinanceTickerResponse) {

	currentPrice, err := convertToFloat64(ticker.CurrentPrice)
	if err != nil {
		log.Printf("Failed to convert CurrentPrice: %v", err)
		return
	}
	volume, err := convertToFloat64(ticker.Volume)
	if err != nil {
		log.Printf("Failed to convert Volume: %v", err)
		return
	}

	symbol := strings.ToLower(ticker.Symbol)
	coinID := symbolToCoinGeckoID[symbol]
	if coinID == "" {
		log.Printf("No CoinGecko ID mapping for %s", ticker.Symbol)
		return
	}

	// Kiểm tra và cập nhật dữ liệu tháng
	monthData, exists := monthlyData[symbol]
	currentTime := time.UnixMilli(ticker.EventTime)
	monthStart := getMonthStart(currentTime)

	if !exists || monthData.StartTime != monthStart {
		// Khởi tạo dữ liệu tháng mới
		monthData = initializeMonthlyData(currentPrice, ticker.EventTime)
		monthlyData[symbol] = monthData
	} else {
		// Cập nhật dữ liệu tháng
		monthData.ClosePrice = currentPrice
		if currentPrice > monthData.HighPrice {
			monthData.HighPrice = currentPrice
		}
		if currentPrice < monthData.LowPrice {
			monthData.LowPrice = currentPrice
		}
		monthData.Volume += volume
	}

	// Phần lấy tên coin và tính market cap giữ nguyên
	coinName, err := getCoinName(coinID)
	if err != nil {
		log.Printf("Failed to get coin name for %s: %v", ticker.Symbol, err)
		coinName = strings.ToUpper(strings.ReplaceAll(ticker.Symbol, "USDT", ""))
	}

	marketCap, err := calculateMarketCap(currentPrice, coinID)
	if err != nil {
		log.Printf("Failed to calculate market cap for %s: %v", ticker.Symbol, err)
		return
	}

	// Gửi dữ liệu tháng
	ConnectToSMCMonth(
		ticker.EventTime,
		coinName,
		fmt.Sprintf("%.2f", monthData.OpenPrice),
		fmt.Sprintf("%.2f", monthData.HighPrice),
		fmt.Sprintf("%.2f", monthData.LowPrice),
		fmt.Sprintf("%.2f", monthData.ClosePrice),
		fmt.Sprintf("%.2f", monthData.Volume),
		"0",
		fmt.Sprintf("%.2f", monthData.ClosePrice-monthData.OpenPrice),
		fmt.Sprintf("%.2f", (monthData.ClosePrice-monthData.OpenPrice)/monthData.OpenPrice*100),
		fmt.Sprintf("%.2f", marketCap),
		symbol,
	)
}
