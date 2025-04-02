package narrativesPerforments

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type WeeklyData struct {
	OpenPrice  float64   // Giá mở cửa đầu tuần
	HighPrice  float64   // Giá cao nhất trong tuần
	LowPrice   float64   // Giá thấp nhất trong tuần
	ClosePrice float64   // Giá đóng cửa hiện tại (cập nhật liên tục)
	StartTime  time.Time // Thời gian bắt đầu tuần
	Volume     float64   // Tổng khối lượng giao dịch trong tuần
	Symbol     string    // Ticker của coin
}

// Biến toàn cục để lưu trữ dữ liệu tuần
var weeklyData = make(map[string]*WeeklyData)

func getWeekStart(t time.Time) time.Time {
	// Chuyển về UTC và đặt về 00:00 thứ Hai
	t = t.UTC()
	weekday := t.Weekday()
	if weekday == time.Sunday {
		weekday = 7 // Điều chỉnh để Chủ Nhật là ngày cuối tuần
	}
	daysToSubtract := int(weekday - time.Monday)
	return t.AddDate(0, 0, -daysToSubtract).Truncate(24 * time.Hour)
}

func initializeWeeklyData(currentPrice float64, eventTime int64) *WeeklyData {
	eventTimeUTC := time.UnixMilli(eventTime).UTC()
	weekStart := getWeekStart(eventTimeUTC)
	return &WeeklyData{
		OpenPrice:  currentPrice,
		HighPrice:  currentPrice,
		LowPrice:   currentPrice,
		ClosePrice: currentPrice,
		StartTime:  weekStart,
		Volume:     0,
	}
}

func RecordWeek(ticker BinanceTickerResponse) {
	// Chuyển đổi các giá trị cần thiết
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

	// Kiểm tra và cập nhật dữ liệu tuần
	weekData, exists := weeklyData[symbol]
	currentTime := time.UnixMilli(ticker.EventTime)
	weekStart := getWeekStart(currentTime)

	if !exists || weekData.StartTime != weekStart {
		// Khởi tạo dữ liệu tuần mới
		weekData = initializeWeeklyData(currentPrice, ticker.EventTime)
		weeklyData[symbol] = weekData
	} else {
		// Cập nhật dữ liệu tuần
		weekData.ClosePrice = currentPrice
		if currentPrice > weekData.HighPrice {
			weekData.HighPrice = currentPrice
		}
		if currentPrice < weekData.LowPrice {
			weekData.LowPrice = currentPrice
		}
		weekData.Volume += volume // Cộng dồn volume
	}

	// Lấy tên coin
	coinName, err := getCoinName(coinID)
	if err != nil {
		log.Printf("Failed to get coin name for %s: %v", ticker.Symbol, err)
		coinName = strings.ToUpper(strings.ReplaceAll(ticker.Symbol, "USDT", ""))
	}

	// Tính vốn hóa thị trường
	marketCap, err := calculateMarketCap(currentPrice, coinID)
	if err != nil {
		log.Printf("Failed to calculate market cap for %s: %v", ticker.Symbol, err)
		return
	}

	ConnectToSMCWeek(
		ticker.EventTime,
		coinName,
		fmt.Sprintf("%.2f", weekData.OpenPrice),
		fmt.Sprintf("%.2f", weekData.HighPrice),
		fmt.Sprintf("%.2f", weekData.LowPrice),
		fmt.Sprintf("%.2f", weekData.ClosePrice),
		fmt.Sprintf("%.2f", weekData.Volume),
		"0",
		fmt.Sprintf("%.2f", weekData.ClosePrice-weekData.OpenPrice),
		fmt.Sprintf("%.2f", (weekData.ClosePrice-weekData.OpenPrice)/weekData.OpenPrice*100),
		fmt.Sprintf("%.2f", marketCap),
		symbol,
	)
}
