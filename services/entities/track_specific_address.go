package entities

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TrackMultipleAddresses theo dõi nhiều địa chỉ ví cùng lúc
func TrackMultipleAddresses(apiKey string) {
	log.Printf("[BẮT ĐẦU] Theo dõi %d địa chỉ cụ thể", len(TargetAddresses))

	// Khởi tạo dữ liệu ban đầu cho mỗi địa chỉ
	for _, address := range TargetAddresses {
		monitorMutex.Lock()
		monitorDataMap[address] = &AddressMonitorData{
			Address:     address,
			LastUpdated: time.Now(),
		}
		monitorMutex.Unlock()

		// Tạo goroutine riêng cho mỗi địa chỉ để theo dõi
		go trackSingleAddress(apiKey, address)
	}

	// Chờ vô thời hạn
	select {}
}

// trackSingleAddress theo dõi một địa chỉ ví cụ thể
func trackSingleAddress(apiKey string, address string) {
	log.Printf("[THEO DÕI] Bắt đầu theo dõi địa chỉ: %s", address)

	// Cập nhật dữ liệu ban đầu
	updateAddressData(apiKey, address)

	// Bắt đầu một goroutine để theo dõi giao dịch mới
	go monitorAddressTransactions(apiKey, address)

	// Cập nhật dữ liệu theo định kỳ
	for {
		time.Sleep(MonitoringInterval)
		updateAddressData(apiKey, address)
	}
}

// GetAddressBalance lấy số dư BTCB của một địa chỉ với cơ chế retry
func GetAddressBalance(apiKey string, address string) (float64, error) {
	maxRetries := 5
	retryDelay := 3 * time.Second

	for i := 0; i < maxRetries; i++ {
		url := fmt.Sprintf("%s?module=account&action=tokenbalance&contractaddress=%s&address=%s&tag=latest&apikey=%s",
			BscScanAPIBaseURL, BTCBTokenAddress, address, apiKey)

		resp, err := http.Get(url)
		if err != nil {
			if i < maxRetries-1 {
				time.Sleep(retryDelay)
				retryDelay *= 2
				continue
			}
			return 0, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			if i < maxRetries-1 {
				time.Sleep(retryDelay)
				retryDelay *= 2
				continue
			}
			return 0, err
		}

		var response TokenBalanceResponse
		if err := json.Unmarshal(body, &response); err != nil {
			if i < maxRetries-1 {
				time.Sleep(retryDelay)
				retryDelay *= 2
				continue
			}
			return 0, err
		}

		if response.Status != "1" {
			if response.Message == "Max rate limit reached" ||
				strings.Contains(response.Message, "rate limit") ||
				response.Message == "NOTOK" {
				if i < maxRetries-1 {
					time.Sleep(retryDelay)
					retryDelay *= 2
					continue
				}
			}
			return 0, fmt.Errorf("API trả về lỗi: %s", response.Message)
		}

		// Chuyển đổi từ wei sang BTCB
		balanceBTCB, err := convertWeiToToken(response.Result, BTCBDecimals)
		if err != nil {
			return 0, err
		}

		// Chuyển sang float
		balanceFloat, err := strconv.ParseFloat(balanceBTCB, 64)
		if err != nil {
			return 0, err
		}

		return balanceFloat, nil
	}

	return 0, fmt.Errorf("đã vượt quá số lần thử lại tối đa")
}

// updateAddressData cập nhật dữ liệu của một địa chỉ
func updateAddressData(apiKey string, address string) {
	// Lấy số dư cũ để so sánh
	oldBalance := 0.0

	monitorMutex.RLock()
	if data, exists := monitorDataMap[address]; exists {
		oldBalance = data.Balance
	}
	monitorMutex.RUnlock()

	// Lấy số dư BTCB
	balance, err := GetAddressBalance(apiKey, address)
	if err != nil {
		return
	}

	// Tính phần trăm nắm giữ
	percentage := CalculateHoldingPercentage(balance)

	// Tính giá trị USD
	usdValue := balance * currentBTCBPrice

	// Cập nhật dữ liệu theo dõi
	monitorMutex.Lock()
	defer monitorMutex.Unlock()

	data, exists := monitorDataMap[address]
	if !exists {
		return
	}

	// Kiểm tra sự thay đổi số dư
	balanceChange := balance - oldBalance

	// Cập nhật dữ liệu
	data.Balance = balance
	data.USDValue = usdValue
	data.Percentage = percentage
	data.LastUpdated = time.Now()

	// Ghi log nếu có sự thay đổi số dư đáng kể (sử dụng giá trị tuyệt đối để bắt cả tăng và giảm)
	if math.Abs(balanceChange) > 0.00001 {
		var addressType string
		if _, ok := exchangeAddresses[address]; ok {
			addressType = "Sàn giao dịch"
		} else if _, ok := organizationAddresses[address]; ok {
			addressType = "Tổ chức"
		} else {
			addressType = "Cá nhân"
		}

		changeDirection := "tăng"
		changeValue := balanceChange
		if balanceChange < 0 {
			changeDirection = "giảm"
			changeValue = -balanceChange
		}

		log.Printf("[THAY ĐỔI] %s %s: Số dư %s %.8f BTCB (%.2f USD). Số dư hiện tại: %.8f BTCB (%.2f USD, %.4f%%)",
			address, addressType, changeDirection, changeValue, changeValue*currentBTCBPrice,
			balance, usdValue, percentage)
	}
}

// monitorAddressTransactions theo dõi các giao dịch mới của địa chỉ
func monitorAddressTransactions(apiKey string, address string) {
	api := NewBscScanAPI(apiKey)
	lastCheckTime := time.Now().Unix()

	for {
		time.Sleep(30 * time.Second) // Tăng thời gian giữa các lần kiểm tra giao dịch để tránh rate limit

		// Lấy các giao dịch liên quan đến địa chỉ
		transactions, err := api.GetAddressTransactions(address, lastCheckTime)
		if err != nil {
			continue
		}

		if len(transactions) > 0 {
			log.Printf("[GIAO DỊCH] Phát hiện %d giao dịch mới cho địa chỉ %s", len(transactions), address)

			// Cập nhật thời gian kiểm tra
			lastCheckTime = time.Now().Unix()

			// Thêm độ trễ để đảm bảo số dư được cập nhật trên blockchain
			time.Sleep(5 * time.Second)

			// Xử lý các giao dịch mới
			for _, tx := range transactions {
				txValueBTCB, _ := convertWeiToToken(tx.Value, BTCBDecimals)
				txValueBTCBFloat, _ := strconv.ParseFloat(txValueBTCB, 64)

				isInflow := tx.To == address
				flowType := "NHẬN VÀO"
				if !isInflow {
					flowType = "GỬI RA"
				}

				fromTo := ""
				if isInflow {
					fromTo = "từ " + tx.From
				} else {
					fromTo = "đến " + tx.To
				}

				log.Printf("[GIAO DỊCH] %s [%s]: %s %s %.8f BTCB",
					address,
					tx.Hash,
					flowType,
					fromTo,
					txValueBTCBFloat)

				// Cập nhật danh sách giao dịch
				timestamp := time.Unix(tx.TimeStamp, 0)

				monitorMutex.Lock()
				monitorDataMap[address].Transactions = append(monitorDataMap[address].Transactions, TransactionInfo{
					Hash:      tx.Hash,
					Timestamp: timestamp,
					From:      tx.From,
					To:        tx.To,
					Value:     txValueBTCBFloat,
					IsInflow:  isInflow,
				})

				// Giới hạn số lượng giao dịch lưu trữ (giữ 50 giao dịch gần nhất)
				if len(monitorDataMap[address].Transactions) > 50 {
					// Cắt bỏ giao dịch cũ nhất
					monitorDataMap[address].Transactions = monitorDataMap[address].Transactions[len(monitorDataMap[address].Transactions)-50:]
				}
				monitorMutex.Unlock()
			}

			// Cập nhật lại số dư sau khi có giao dịch mới
			updateAddressData(apiKey, address)
		}
	}
}

// GetMonitorData trả về dữ liệu theo dõi cho một địa chỉ cụ thể
func GetMonitorData(address string) (AddressMonitorData, bool) {
	monitorMutex.RLock()
	defer monitorMutex.RUnlock()

	data, exists := monitorDataMap[address]
	if !exists {
		return AddressMonitorData{}, false
	}

	return *data, true
}

// GetAllMonitoredAddresses trả về danh sách tất cả các địa chỉ đang được theo dõi
func GetAllMonitoredAddresses() []string {
	return TargetAddresses
}

// AddAddressToMonitor thêm một địa chỉ mới vào danh sách theo dõi
func AddAddressToMonitor(apiKey string, address string) {
	// Kiểm tra xem địa chỉ đã tồn tại trong danh sách chưa
	for _, addr := range TargetAddresses {
		if addr == address {
			return
		}
	}

	// Thêm địa chỉ vào danh sách
	TargetAddresses = append(TargetAddresses, address)

	// Khởi tạo dữ liệu theo dõi cho địa chỉ mới
	monitorMutex.Lock()
	monitorDataMap[address] = &AddressMonitorData{
		Address:     address,
		LastUpdated: time.Now(),
	}
	monitorMutex.Unlock()

	// Bắt đầu theo dõi địa chỉ mới
	go trackSingleAddress(apiKey, address)

	log.Printf("[THÊM] Đã thêm địa chỉ %s vào danh sách theo dõi", address)
}

// GetAddressTransactions lấy các giao dịch của một địa chỉ từ BscScan với xử lý lỗi cải tiến
func (api *BscScanAPI) GetAddressTransactions(address string, startTime int64) ([]Transaction, error) {
	maxRetries := 5
	retryDelay := 3 * time.Second

	for i := 0; i < maxRetries; i++ {
		url := fmt.Sprintf("%s?module=account&action=tokentx&contractaddress=%s&address=%s&startblock=0&endblock=99999999&sort=desc&apikey=%s",
			BscScanAPIBaseURL, BTCBTokenAddress, address, api.APIKey)

		resp, err := api.RateLimitedRequest(url)
		if err != nil {
			if i < maxRetries-1 {
				time.Sleep(retryDelay)
				retryDelay *= 2
				continue
			}
			return nil, err
		}

		// Sử dụng RawMessage để xử lý linh hoạt hơn
		var response struct {
			Status  string          `json:"status"`
			Message string          `json:"message"`
			Result  json.RawMessage `json:"result"`
		}

		if err := json.Unmarshal(resp, &response); err != nil {
			if i < maxRetries-1 {
				time.Sleep(retryDelay)
				retryDelay *= 2
				continue
			}
			return nil, err
		}

		if response.Status != "1" {
			if response.Message == "No transactions found" {
				return []Transaction{}, nil
			}

			if response.Message == "Max rate limit reached" ||
				strings.Contains(response.Message, "rate limit") ||
				response.Message == "NOTOK" {
				if i < maxRetries-1 {
					time.Sleep(retryDelay)
					retryDelay *= 2
					continue
				}
			}
			return nil, fmt.Errorf("API trả về lỗi: %s", response.Message)
		}

		// Kiểm tra xem Result có phải là mảng hay không
		if len(response.Result) > 0 && response.Result[0] == '[' {
			var transactions []Transaction
			if err := json.Unmarshal(response.Result, &transactions); err != nil {
				if i < maxRetries-1 {
					time.Sleep(retryDelay)
					retryDelay *= 2
					continue
				}
				return nil, err
			}

			// Lọc các giao dịch sau startTime và chỉ lấy các giao dịch BTCB có giá trị đáng kể
			var newTransactions []Transaction
			for _, tx := range transactions {
				if tx.TimeStamp > startTime && tx.TokenSymbol == "BTCB" {
					// Kiểm tra giá trị giao dịch
					txValueBTCB, err := convertWeiToToken(tx.Value, BTCBDecimals)
					if err != nil {
						continue
					}

					txValueBTCBFloat, err := strconv.ParseFloat(txValueBTCB, 64)
					if err != nil {
						continue
					}

					// Chỉ lấy các giao dịch có giá trị đáng kể (ví dụ: > 0.0001 BTCB)
					if txValueBTCBFloat > 0.0001 {
						newTransactions = append(newTransactions, tx)
					}
				}
			}

			return newTransactions, nil
		} else {
			// Trường hợp Result là string hoặc định dạng khác
			return []Transaction{}, nil
		}
	}

	return nil, fmt.Errorf("đã vượt quá số lần thử lại tối đa")
}
