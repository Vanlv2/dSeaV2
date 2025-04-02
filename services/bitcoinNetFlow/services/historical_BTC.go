package services

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/beldur/kraken-go-api-client"
)

// Biến global để lưu trữ dữ liệu lịch sử BTC
var historicalBTCOrders []BTCOrder
var historicalBTCOrdersMutex sync.RWMutex
var historicalBTCOrdersByTime map[int64][]BTCOrder
var historicalBTCOrdersByTimeMutex sync.RWMutex

// GetHistoricalBTCOrders lấy dữ liệu giao dịch lịch sử BTC
func GetHistoricalBTCOrders() []BTCOrder {
	historicalBTCOrdersMutex.RLock()
	defer historicalBTCOrdersMutex.RUnlock()

	// Tạo bản sao của dữ liệu để tránh race condition
	ordersCopy := make([]BTCOrder, len(historicalBTCOrders))
	copy(ordersCopy, historicalBTCOrders)

	return ordersCopy
}

// GetHistoricalBTCOrdersSorted trả về danh sách các lệnh lịch sử đã được sắp xếp theo thời gian
func GetHistoricalBTCOrdersSorted() []BTCOrder {
	historicalBTCOrdersByTimeMutex.RLock()
	defer historicalBTCOrdersByTimeMutex.RUnlock()

	// Lấy tất cả các timestamp và sắp xếp chúng
	var timestamps []int64
	for ts := range historicalBTCOrdersByTime {
		timestamps = append(timestamps, ts)
	}
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i] < timestamps[j]
	})

	// Tạo danh sách các lệnh đã sắp xếp
	var sortedOrders []BTCOrder
	for _, ts := range timestamps {
		sortedOrders = append(sortedOrders, historicalBTCOrdersByTime[ts]...)
	}

	return sortedOrders
}

// HandleHistoricalBTCData thu thập dữ liệu lịch sử BTC từ Kraken
func HandleHistoricalBTCData() {
	// Khởi tạo các biến global
	historicalBTCOrders = []BTCOrder{}
	historicalBTCOrdersByTime = make(map[int64][]BTCOrder)

	// Tạo client Kraken API không cần API key/secret cho Public API
	api := krakenapi.New("", "")

	// Xác định cặp tiền tệ bạn muốn xem giao dịch
	pair := "XBTUSDT"

	var wg sync.WaitGroup
	wg.Add(1)

	// Khởi chạy goroutine để xử lý việc lấy dữ liệu
	go func() {
		defer wg.Done()

		// Lấy thời gian hiện tại làm điểm bắt đầu
		currentTimestamp := time.Now().Unix()

		noDataCount := 0
		totalTradesCollected := 0 // Biến đếm tổng số giao dịch đã lấy

		// Map để theo dõi các giao dịch đã xử lý, tránh trùng lặp
		processedTrades := make(map[string]bool)

		for {
			// Gọi API để lấy dữ liệu giao dịch
			result, err := api.Trades(pair, currentTimestamp)
			if err != nil {
				break
			}

			if len(result.Trades) == 0 {
				// Lùi thời gian về quá khứ với khoảng nhỏ hơn nếu không có dữ liệu
				currentTimestamp -= 6 // Lùi 6 giây
				noDataCount++

				// Nếu không có dữ liệu sau nhiều lần thử, dừng lại
				if noDataCount > 144 { // Thử 144 lần
					break
				}

				continue
			}

			// Đặt lại bộ đếm không có dữ liệu khi tìm thấy dữ liệu
			noDataCount = 0

			// Tìm timestamp cũ nhất trong danh sách giao dịch
			var oldestTime int64 = currentTimestamp
			foundOlder := false

			// Đếm số giao dịch mới trong lần này
			newTradesCount := 0

			for _, trade := range result.Trades {
				// Tạo ID duy nhất cho giao dịch
				tradeID := fmt.Sprintf("%d-%s-%s-%t-%t",
					trade.Time,
					trade.Price,
					trade.Volume,
					trade.Buy,
					trade.Market)

				// Kiểm tra xem giao dịch này đã được xử lý chưa
				if _, exists := processedTrades[tradeID]; exists {
					continue
				}

				// Đánh dấu giao dịch này đã được xử lý
				processedTrades[tradeID] = true
				newTradesCount++

				tradeTime := time.Unix(trade.Time, 0)

				// Chuyển đổi dữ liệu từ string sang float64
				price, err := strconv.ParseFloat(trade.Price, 64)
				if err != nil {
					continue
				}

				volume, err := strconv.ParseFloat(trade.Volume, 64)
				if err != nil {
					continue
				}

				// Xác định phía giao dịch (mua/bán) cho cấu trúc Order
				sideEng := "sell"
				if trade.Buy {
					sideEng = "buy"
				}

				// Tạo đối tượng Order mới
				order := BTCOrder{
					Timestamp: tradeTime,
					OrderID:   tradeID,
					Symbol:    "BTC", // XBTUSDT tương đương với BTC
					Side:      sideEng,
					Amount:    volume,
					Price:     price,
					Source:    "Historical",
				}

				// Thêm vào biến global với khóa mutex
				historicalBTCOrdersMutex.Lock()
				historicalBTCOrders = append(historicalBTCOrders, order)
				historicalBTCOrdersMutex.Unlock()

				// Thêm vào map theo thời gian
				historicalBTCOrdersByTimeMutex.Lock()
				if _, exists := historicalBTCOrdersByTime[trade.Time]; !exists {
					historicalBTCOrdersByTime[trade.Time] = []BTCOrder{}
				}
				historicalBTCOrdersByTime[trade.Time] = append(historicalBTCOrdersByTime[trade.Time], order)
				historicalBTCOrdersByTimeMutex.Unlock()

				// Cập nhật timestamp cũ nhất nếu giao dịch này cũ hơn
				if trade.Time < oldestTime {
					oldestTime = trade.Time
					foundOlder = true
				}
			}

			// Cập nhật tổng số giao dịch đã lấy (chỉ tính giao dịch mới)
			totalTradesCollected += newTradesCount

			// Nếu không tìm thấy giao dịch cũ hơn, có thể là đã đạt đến giới hạn
			if !foundOlder {
				currentTimestamp -= 1000 // Lùi thêm 1000 giây
			} else {
				// Sử dụng thời gian cũ nhất làm điểm bắt đầu cho lần sau
				// Lùi thêm 1 giây để đảm bảo không bỏ sót giao dịch
				currentTimestamp = oldestTime - 1
			}

			// Chờ một chút trước khi gọi API tiếp để tránh bị giới hạn tần suất gọi
			time.Sleep(2 * time.Second)
		}
	}()

	// Đợi goroutine hoàn thành
	wg.Wait()
}

// ProcessHistoricalBTCData xử lý dữ liệu lịch sử BTC để sử dụng trong models
func ProcessHistoricalBTCData() []BTCOrder {
	// Lấy dữ liệu đã sắp xếp theo thời gian
	sortedOrders := GetHistoricalBTCOrdersSorted()

	// Tạo bản sao để tránh race condition
	ordersCopy := make([]BTCOrder, len(sortedOrders))
	copy(ordersCopy, sortedOrders)

	return ordersCopy
}
