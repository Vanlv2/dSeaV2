package bitcoinNetFlow

import (
	"time"

	"main/services/bitcoinNetFlow/services"
)

// RealTimeFlowData cấu trúc dữ liệu để lưu trữ thông tin dòng tiền BTC thời gian thực
type RealTimeFlowData struct {
	Timestamp time.Time
	Incoming  map[string]float64
	Outgoing  map[string]float64
	Balance   map[string]float64
	Source    string // "RealTime" hoặc "Historical"
}

// Biến global để lưu trữ kết quả
var realTimeResult map[time.Time]RealTimeFlowData

// AnalyzeRealTimeBTCFlow phân tích dòng tiền BTC thời gian thực
func AnalyzeRealTimeBTCFlow(timeInterval int, initialBalance float64) (map[time.Time]RealTimeFlowData, error) {

	// Kết quả sẽ được lưu trong map này
	result := make(map[time.Time]RealTimeFlowData)

	// Lấy thời gian hiện tại
	now := time.Now()

	// Lấy dữ liệu thời gian thực
	allOrders := services.GetRealTimeBTCOrders()

	// Lọc chỉ lấy các lệnh BTC
	var btcOrders []services.BTCOrder
	for _, order := range allOrders {
		if order.Symbol == "BTC" {
			btcOrders = append(btcOrders, order)
		}
	}

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
		flowData := RealTimeFlowData{
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

// PerformRealTimeAnalysis thực hiện phân tích dữ liệu BTC thời gian thực
func PerformRealTimeAnalysis(timeInterval int, initialBalance float64) {
	// Phân tích dòng tiền BTC và cập nhật biến global result cho dữ liệu thời gian thực
	newResult, _ := AnalyzeRealTimeBTCFlow(timeInterval, initialBalance)

	// Cập nhật biến global realTimeResult
	realTimeResult = newResult
}

// StartPeriodicRealTimeBTCAnalysis định kỳ phân tích dữ liệu BTC thời gian thực
func StartPeriodicRealTimeBTCAnalysis(interval time.Duration, timeInterval int, initialBalance float64) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Phân tích lần đầu ngay khi khởi động
		PerformRealTimeAnalysis(timeInterval, initialBalance)

		// Sau đó phân tích theo chu kỳ
		for range ticker.C {
			PerformRealTimeAnalysis(timeInterval, initialBalance)
		}
	}()
}

// HandleRealTimeTrading phân tích dữ liệu giao dịch thời gian thực và trả về kết quả
func HandleRealTimeTrading(timeSecond int) map[time.Time]RealTimeFlowData {
	// Thiết lập các tham số
	initialBalance := 100000.0

	// Phân tích dữ liệu và trả về kết quả
	result, _ := AnalyzeRealTimeBTCFlow(timeSecond, initialBalance)
	return result
}

// GetRealTimeAnalysisResult lấy kết quả phân tích thời gian thực
func GetRealTimeAnalysisResult() map[time.Time]RealTimeFlowData {
	resultMap := make(map[time.Time]RealTimeFlowData)
	for t, data := range realTimeResult {
		flowData := RealTimeFlowData{
			Timestamp: data.Timestamp,
			Incoming:  data.Incoming,
			Outgoing:  data.Outgoing,
			Balance:   data.Balance,
		}
		resultMap[t] = flowData
	}
	return resultMap
}

// GetRealTimeOriginalResult lấy kết quả phân tích gốc thời gian thực
func GetRealTimeOriginalResult() map[time.Time]RealTimeFlowData {
	return realTimeResult
}
