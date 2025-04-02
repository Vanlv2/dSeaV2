package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Thông tin tổng cung BTCB
type BTCBSupplyInfo struct {
	TotalSupply float64
	LastUpdated time.Time
}

var btcbSupplyInfo BTCBSupplyInfo

// GetBTCBTotalSupply lấy tổng cung của BTCB từ API
func GetBTCBTotalSupply() (float64, error) {
	// Sử dụng API BSCScan để lấy tổng cung
	url := fmt.Sprintf("%s?module=stats&action=tokensupply&contractaddress=%s&apikey=%s",
		BscScanAPIBaseURL, BTCBTokenAddress, "TZR6PYQJPSREBUJHTYWP948TXHD3MXNQ7W")

	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("lỗi khi gọi API để lấy tổng cung: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("lỗi khi đọc dữ liệu từ API: %v", err)
	}

	var response struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("lỗi khi phân tích dữ liệu: %v", err)
	}

	if response.Status != "1" {
		return 0, fmt.Errorf("API trả về lỗi: %s", response.Message)
	}

	// Chuyển đổi từ wei sang BTCB
	totalSupply, err := convertWeiToToken(response.Result, BTCBDecimals)
	if err != nil {
		return 0, fmt.Errorf("lỗi khi chuyển đổi tổng cung: %v", err)
	}

	// Chuyển sang float
	totalSupplyFloat, err := strconv.ParseFloat(totalSupply, 64)
	if err != nil {
		return 0, fmt.Errorf("lỗi khi chuyển đổi tổng cung sang số: %v", err)
	}

	return totalSupplyFloat, nil
}

// Tính phần trăm nắm giữ so với tổng cung
func CalculateHoldingPercentage(balance float64) float64 {
	if btcbSupplyInfo.TotalSupply <= 0 {
		return 0
	}
	return (balance / btcbSupplyInfo.TotalSupply) * 100
}

// Cập nhật thông tin tổng cung BTCB định kỳ
func UpdateBTCBSupplyRegularly() {
	for {
		supply, err := GetBTCBTotalSupply()
		if err != nil {
			log.Printf("Cảnh báo: Không thể cập nhật thông tin tổng cung BTCB: %v", err)
		} else {
			btcbSupplyInfo.TotalSupply = supply
			btcbSupplyInfo.LastUpdated = time.Now()
			log.Printf("Đã cập nhật tổng cung BTCB: %.8f", supply)
		}

		// Cập nhật mỗi 1 giờ
		time.Sleep(1 * time.Hour)
	}
}

func main() {
	// Cấu hình log để chỉ in ra terminal
	log.SetFlags(log.LstdFlags)
	
	log.Println("=== Khởi động chương trình quét số dư BTCB và quy đổi USD ===")
	log.Printf("Ngưỡng số dư tối thiểu cho tổ chức: %.2f BTCB\n", OrganizationThreshold)
	log.Printf("Ngưỡng số dư tối thiểu cho sàn giao dịch: %.2f BTCB\n", ExchangeThreshold)

	// Lấy tổng cung BTCB ban đầu
	totalSupply, err := GetBTCBTotalSupply()
	if err != nil {
		log.Printf("Cảnh báo: Không thể lấy tổng cung BTCB ban đầu: %v", err)
	} else {
		btcbSupplyInfo.TotalSupply = totalSupply
		btcbSupplyInfo.LastUpdated = time.Now()
		log.Printf("Tổng cung BTCB ban đầu: %.8f", totalSupply)
	}

	// Bắt đầu goroutine để cập nhật tổng cung BTCB định kỳ
	go UpdateBTCBSupplyRegularly()

	// Gọi hàm xử lý chính từ handle_value.go
	handleValueAddress()
}
