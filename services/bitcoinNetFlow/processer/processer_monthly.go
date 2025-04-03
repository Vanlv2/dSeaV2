package processer

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	caculator "main/services/bitcoinNetFlow/caculator_datas"
	services "main/services/bitcoinNetFlow/services"
	sendData "main/services/bitcoinNetFlow/smart_contract/send_data"
)

// Các constants cho xử lý tháng
var (
	monthlyAnalysisInterval    = 30 * 24 * time.Hour // Phân tích 1 lần mỗi tháng (xấp xỉ)
	monthlyTimeSegmentInterval = 2592000             // Khoảng thời gian phân tích 1 tháng (30*24*3600 giây)
)


func Handle_monthly_SMC() {
	// Tạo channel để xử lý tín hiệu dừng
	stopChan := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// WaitGroup để đảm bảo tất cả goroutines đều kết thúc an toàn
	var wg sync.WaitGroup

	// Goroutine xử lý tín hiệu dừng
	go func() {
		<-sigChan
		close(stopChan)
	}()

	// 1. KHỞI ĐỘNG CÁC GOROUTINE LẤY DỮ LIỆU
	// 1.1 Khởi động thu thập dữ liệu từ nguồn thời gian thực
	wg.Add(1)
	go func() {
		defer wg.Done()
		configFile := "config_chain/binance.json"
		services.HandleRealTimeBTC(configFile, stopChan)
	}()

	// 1.2 Khởi động thu thập dữ liệu từ nguồn lịch sử
	wg.Add(1)
	go func() {
		defer wg.Done()
		services.HandleHistoricalBTCData()
	}()

	// Chờ 5 giây để đảm bảo dữ liệu bắt đầu được thu thập
	time.Sleep(5 * time.Second)

	// 2. KHỞI ĐỘNG CÁC GOROUTINE XỬ LÝ DỮ LIỆU VÀ GỬI LÊN BLOCKCHAIN
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(monthlyAnalysisInterval)
		defer ticker.Stop()

		// Phân tích và gửi lần đầu ngay lập tức
		processAndSendMonthlyDataToBlockchain(monthlyTimeSegmentInterval)

		// Sau đó phân tích và gửi theo chu kỳ
		for {
			select {
			case <-ticker.C:
				processAndSendMonthlyDataToBlockchain(monthlyTimeSegmentInterval)
			case <-stopChan:
				return
			}
		}
	}()

	// CHỜ TẤT CẢ GOROUTINES KẾT THÚC KHI CÓ TÍN HIỆU DỪNG
	wg.Wait()
}

// Hàm xử lý phân tích dữ liệu và gửi lên blockchain theo tháng
func processAndSendMonthlyDataToBlockchain(timeInterval int) {
	// BƯỚC 1: Phân tích dữ liệu thời gian thực
	realTimeResult := caculator.HandleRealTimeTrading(timeInterval)
	realTimeRecordCount := len(realTimeResult)

	// BƯỚC 2: Phân tích dữ liệu lịch sử
	historicalResult := caculator.HandleHistoricalTrading(timeInterval)
	historicalRecordCount := len(historicalResult)

	// BƯỚC 3: Gửi dữ liệu lên blockchain nếu có dữ liệu để gửi
	if realTimeRecordCount > 0 || historicalRecordCount > 0 {
		// Khởi động goroutine riêng để gửi dữ liệu lên blockchain
		go func() {
			sendData.SendDataToSMC(realTimeResult, historicalResult, "monthly")
		}()
	}
}
