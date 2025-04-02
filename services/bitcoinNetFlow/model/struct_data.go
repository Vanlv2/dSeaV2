package bitcoinNetFlow

import (
	"time"

	services "main/services/bitcoinNetFlow/services"
)

// BTCOrder cấu trúc dữ liệu để lưu trữ thông tin lệnh BTC
type BTCOrder struct {
	Timestamp time.Time
	OrderID   string
	Symbol    string
	Side      string
	Amount    float64
	Price     float64
	Source    string // "RealTime" hoặc "Historical"
}

// BTCFlowData cấu trúc dữ liệu chung cho luồng dữ liệu BTC
type BTCFlowData struct {
	Timestamp time.Time
	Incoming  map[string]float64
	Outgoing  map[string]float64
	Balance   map[string]float64
	Source    string // "RealTime" hoặc "Historical"
}

// ProcessRealTimeBTCData xử lý dữ liệu BTC thời gian thực
func ProcessRealTimeBTCData() BTCFlowData {
	// Khởi tạo FlowData với các map rỗng
	flowData := BTCFlowData{
		Timestamp: time.Now(),
		Incoming:  make(map[string]float64),
		Outgoing:  make(map[string]float64),
		Balance:   make(map[string]float64),
		Source:    "RealTime",
	}

	// Lấy dữ liệu thời gian thực
	realTimeOrders := services.GetRealTimeBTCOrders()
	// Xử lý dữ liệu
	for _, order := range realTimeOrders {
		symbol := order.Symbol

		// Kiểm tra nếu map chưa có key này thì khởi tạo
		if _, exists := flowData.Incoming[symbol]; !exists {
			flowData.Incoming[symbol] = 0
		}
		if _, exists := flowData.Outgoing[symbol]; !exists {
			flowData.Outgoing[symbol] = 0
		}
		if _, exists := flowData.Balance[symbol]; !exists {
			flowData.Balance[symbol] = 0
		}

		// Cập nhật dữ liệu dựa trên loại lệnh
		if order.Side == "buy" {
			flowData.Incoming[symbol] += order.Amount
			flowData.Balance[symbol] += order.Amount
		} else if order.Side == "sell" {
			flowData.Outgoing[symbol] += order.Amount
			flowData.Balance[symbol] -= order.Amount
		}
	}

	return flowData
}

// ProcessHistoricalBTCData xử lý dữ liệu BTC lịch sử
func ProcessHistoricalBTCData() BTCFlowData {
	// Khởi tạo FlowData với các map rỗng
	flowData := BTCFlowData{
		Timestamp: time.Now(),
		Incoming:  make(map[string]float64),
		Outgoing:  make(map[string]float64),
		Balance:   make(map[string]float64),
		Source:    "Historical",
	}

	// Lấy dữ liệu lịch sử
	historicalOrders := services.ProcessHistoricalBTCData()

	// Xử lý dữ liệu
	for _, order := range historicalOrders {
		symbol := order.Symbol

		// Kiểm tra nếu map chưa có key này thì khởi tạo
		if _, exists := flowData.Incoming[symbol]; !exists {
			flowData.Incoming[symbol] = 0
		}
		if _, exists := flowData.Outgoing[symbol]; !exists {
			flowData.Outgoing[symbol] = 0
		}
		if _, exists := flowData.Balance[symbol]; !exists {
			flowData.Balance[symbol] = 0
		}

		// Cập nhật dữ liệu dựa trên loại lệnh
		if order.Side == "buy" {
			flowData.Incoming[symbol] += order.Amount
			flowData.Balance[symbol] += order.Amount
		} else if order.Side == "sell" {
			flowData.Outgoing[symbol] += order.Amount
			flowData.Balance[symbol] -= order.Amount
		}
	}

	return flowData
}

// GetCombinedBTCOrderData lấy dữ liệu kết hợp từ cả nguồn thời gian thực và lịch sử
func GetCombinedBTCOrderData() []services.BTCOrder {
	// Lấy dữ liệu từ cả hai nguồn
	realTimeOrders := services.GetRealTimeBTCOrders()
	historicalOrders := services.ProcessHistoricalBTCData()

	// Trả về kết quả
	allOrders := append(realTimeOrders, historicalOrders...)
	return allOrders
}
