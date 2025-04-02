package bitcoinNetFlow

import (
	"time"

	"main/services/bitcoinNetFlow/services"
)

// HistoricalFlowData cấu trúc dữ liệu để lưu trữ thông tin dòng tiền BTC từ dữ liệu lịch sử
type HistoricalFlowData struct {
	Timestamp time.Time
	Incoming  map[string]float64
	Outgoing  map[string]float64
	Balance   map[string]float64
	Source    string // "RealTime" hoặc "Historical"
}

// Biến global để lưu trữ kết quả phân tích từ dữ liệu lịch sử
var historicalResult map[time.Time]HistoricalFlowData

// AnalyzeHistoricalBTCFlow phân tích dòng tiền BTC từ dữ liệu lịch sử
func AnalyzeHistoricalBTCFlow(timeInterval int, initialBalance float64) (map[time.Time]HistoricalFlowData, error) {
	// Kết quả sẽ được lưu trong map này
	result := make(map[time.Time]HistoricalFlowData)

	// Lấy thời gian hiện tại
	now := time.Now()

	// Lấy dữ liệu lịch sử
	allOrders := services.GetHistoricalBTCOrders()

	// Lọc chỉ lấy các lệnh BTC
	var btcOrders []services.BTCOrder
	for _, order := range allOrders {
		if order.Symbol == "BTC" {
			btcOrders = append(btcOrders, order)
		}
	}

	// Khởi tạo balance map cho BTC
	initialBalanceMap := make(map[string]float64)
	initialBalanceMap["BTC"] = initialBalance

	// Tính toán thời điểm bắt đầu
	var oldestTime time.Time
	if len(btcOrders) > 0 {
		oldestTime = btcOrders[0].Timestamp
		for _, order := range btcOrders {
			if order.Timestamp.Before(oldestTime) {
				oldestTime = order.Timestamp
			}
		}
	} else {
		oldestTime = now.Add(-24 * time.Hour) // Mặc định lấy dữ liệu 24h trước nếu không có lệnh nào
	}

	// Làm tròn thời gian bắt đầu xuống khoảng thời gian
	startTime := oldestTime.Truncate(time.Duration(timeInterval) * time.Second)

	// Tạo các khoảng thời gian và phân tích dữ liệu
	for t := startTime; t.Before(now); t = t.Add(time.Duration(timeInterval) * time.Second) {
		endTime := t.Add(time.Duration(timeInterval) * time.Second)

		// Khởi tạo dữ liệu cho khoảng thời gian này
		flowData := HistoricalFlowData{
			Timestamp: endTime, // Lưu timestamp là thời điểm kết thúc khoảng thời gian
			Incoming:  make(map[string]float64),
			Outgoing:  make(map[string]float64),
			Balance:   make(map[string]float64),
		}

		// Khởi tạo giá trị cho BTC
		flowData.Incoming["BTC"] = 0
		flowData.Outgoing["BTC"] = 0
		flowData.Balance["BTC"] = initialBalance // Sử dụng initialBalance cho mỗi khoảng thời gian

		// Tính toán incoming và outgoing cho khoảng thời gian này
		for _, order := range btcOrders {
			if (order.Timestamp.Equal(t) || order.Timestamp.After(t)) && order.Timestamp.Before(endTime) {
				if order.Side == "buy" {
					flowData.Incoming["BTC"] += order.Amount
				} else if order.Side == "sell" {
					flowData.Outgoing["BTC"] += order.Amount
				}
			}
		}

		// Cập nhật số dư CHỈ cho khoảng thời gian này, không lưu qua các khoảng
		netFlow := flowData.Incoming["BTC"] - flowData.Outgoing["BTC"]
		flowData.Balance["BTC"] = initialBalance + netFlow

		// Chỉ lưu kết quả nếu có giao dịch trong khoảng thời gian này
		if flowData.Incoming["BTC"] > 0 || flowData.Outgoing["BTC"] > 0 {
			result[endTime] = flowData
		}
	}

	return result, nil
}

// PerformHistoricalAnalysis thực hiện phân tích dữ liệu BTC từ dữ liệu lịch sử
func PerformHistoricalAnalysis(timeInterval int, initialBalance float64) {
	// Phân tích dòng tiền BTC và cập nhật biến global result cho dữ liệu lịch sử
	newResult, _ := AnalyzeHistoricalBTCFlow(timeInterval, initialBalance)

	// Cập nhật biến global historicalResult
	historicalResult = newResult
}

// StartPeriodicHistoricalBTCAnalysis định kỳ phân tích dữ liệu BTC từ dữ liệu lịch sử
func StartPeriodicHistoricalBTCAnalysis(interval time.Duration, timeInterval int, initialBalance float64) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Phân tích lần đầu ngay khi khởi động
		PerformHistoricalAnalysis(timeInterval, initialBalance)

		// Sau đó phân tích theo chu kỳ
		for range ticker.C {
			PerformHistoricalAnalysis(timeInterval, initialBalance)
		}
	}()
}

// HandleHistoricalTrading phân tích dữ liệu giao dịch từ dữ liệu lịch sử và trả về kết quả
func HandleHistoricalTrading(timeSecond int) map[time.Time]HistoricalFlowData {
	// Thiết lập các tham số
	initialBalance := 100000.0

	// Phân tích dữ liệu và trả về kết quả
	result, _ := AnalyzeHistoricalBTCFlow(timeSecond, initialBalance)
	return result
}

// GetHistoricalAnalysisResult lấy kết quả phân tích từ dữ liệu lịch sử
func GetHistoricalAnalysisResult() map[time.Time]HistoricalFlowData {
	resultMap := make(map[time.Time]HistoricalFlowData)
	for t, data := range historicalResult {
		flowData := HistoricalFlowData{
			Timestamp: data.Timestamp,
			Incoming:  data.Incoming,
			Outgoing:  data.Outgoing,
			Balance:   data.Balance,
		}
		resultMap[t] = flowData
	}
	return resultMap
}

// GetHistoricalOriginalResult lấy kết quả phân tích gốc từ dữ liệu lịch sử
func GetHistoricalOriginalResult() map[time.Time]HistoricalFlowData {
	return historicalResult
}
