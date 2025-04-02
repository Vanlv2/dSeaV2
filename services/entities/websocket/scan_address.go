package main

import (
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"strconv"
	"time"
)

// BTCBDecimals là số chữ số thập phân cho token BTCB
const BTCBDecimals = 18

// Ngưỡng để phân loại địa chỉ
const ExchangeThreshold = 1000.0    // 1000 BTCB cho sàn giao dịch
const OrganizationThreshold = 100.0 // 100 BTCB cho tổ chức

// Danh sách toàn bộ địa chỉ đã thu thập
var allAddresses []string

// Cấu trúc dữ liệu để lưu trữ thông tin địa chỉ
type AddressInfo struct {
	Balance    float64
	Percentage float64
}

// Khai báo lại các biến lưu trữ theo phân loại mới
var highBalanceAddresses = make(map[string]AddressInfo)  // Tất cả địa chỉ có số dư lớn
var organizationAddresses = make(map[string]AddressInfo) // Địa chỉ tổ chức (100-1000 BTCB)
var exchangeAddresses = make(map[string]AddressInfo)     // Địa chỉ sàn giao dịch (>1000 BTCB)

// convertWeiToToken chuyển đổi giá trị wei sang giá trị token dễ đọc
func convertWeiToToken(weiValue string, decimals int) (string, error) {
	wei, ok := new(big.Int).SetString(weiValue, 10)
	if !ok {
		return "", fmt.Errorf("giá trị wei không hợp lệ: %s", weiValue)
	}

	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))

	weiFloat := new(big.Float).SetInt(wei)
	tokenValue := new(big.Float).Quo(weiFloat, divisor)

	// Định dạng với 8 chữ số thập phân (tiêu chuẩn cho BTC)
	result := tokenValue.Text('f', 8)
	return result, nil
}

// CollectAllAddresses thu thập tất cả các địa chỉ từ các giao dịch BTCB
func CollectAllAddresses(apiKey string) ([]string, error) {
	api := NewBscScanAPI(apiKey)

	// Lấy địa chỉ từ các giao dịch BTCB gần đây
	addresses, err := api.GetRecentBTCBAddresses()
	if err != nil {
		return nil, fmt.Errorf("lỗi khi lấy địa chỉ BTCB: %v", err)
	}

	log.Printf("Đã thu thập được %d địa chỉ duy nhất từ các giao dịch BTCB\n", len(addresses))
	return addresses, nil
}

// GetRandomAddresses lấy ngẫu nhiên một số lượng địa chỉ từ danh sách
func GetRandomAddresses(addresses []string, count int) []string {
	if len(addresses) <= count {
		return addresses
	}

	// Tạo một bản sao của mảng địa chỉ để không làm thay đổi mảng gốc
	shuffled := make([]string, len(addresses))
	copy(shuffled, addresses)

	// Xáo trộn mảng
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	// Lấy count địa chỉ đầu tiên
	return shuffled[:count]
}

// UpdateHighBalanceAddresses cập nhật danh sách các địa chỉ có số dư lớn
func UpdateHighBalanceAddresses(address string, balance float64) {
	percentHold := CalculateHoldingPercentage(balance)

	// Cập nhật highBalanceAddresses cho tất cả địa chỉ có số dư lớn
	if balance >= OrganizationThreshold {
		highBalanceAddresses[address] = AddressInfo{
			Balance:    balance,
			Percentage: percentHold,
		}

		// Phân loại theo ngưỡng mới
		if balance >= ExchangeThreshold {
			// Địa chỉ sàn giao dịch (>1000 BTCB)
			exchangeAddresses[address] = AddressInfo{
				Balance:    balance,
				Percentage: percentHold,
			}
		} else {
			// Địa chỉ tổ chức (100-1000 BTCB)
			organizationAddresses[address] = AddressInfo{
				Balance:    balance,
				Percentage: percentHold,
			}
		}
	}
}

// GetHighBalanceAddresses lấy danh sách các địa chỉ có số dư lớn
func GetHighBalanceAddresses() []string {
	var addresses []string
	for addr := range highBalanceAddresses {
		addresses = append(addresses, addr)
	}
	return addresses
}

// MonitorNewTransactions theo dõi các giao dịch mới để cập nhật danh sách địa chỉ
func MonitorNewTransactions(apiKey string) {
	api := NewBscScanAPI(apiKey)
	lastCheckTime := time.Now().Unix()

	for {
		time.Sleep(30 * time.Second) // Kiểm tra giao dịch mới mỗi 30 giây

		log.Println("Kiểm tra các giao dịch mới...")

		transactions, err := api.GetLatestTransactions(lastCheckTime)
		if err != nil {
			log.Printf("Lỗi khi lấy giao dịch mới: %v", err)
			continue
		}

		if len(transactions) > 0 {
			log.Printf("Tìm thấy %d giao dịch mới", len(transactions))

			// Cập nhật thời gian kiểm tra cuối cùng
			lastCheckTime = time.Now().Unix()

			// Thu thập các địa chỉ mới từ giao dịch
			newAddresses := make(map[string]bool)
			for _, tx := range transactions {
				newAddresses[tx.From] = true
				newAddresses[tx.To] = true
			}

			// Thêm các địa chỉ mới vào danh sách
			for addr := range newAddresses {
				// Kiểm tra xem địa chỉ đã có trong danh sách chưa
				found := false
				for _, existingAddr := range allAddresses {
					if existingAddr == addr {
						found = true
						break
					}
				}

				if !found {
					allAddresses = append(allAddresses, addr)
				}
			}

			log.Printf("Đã cập nhật danh sách địa chỉ, tổng số: %d", len(allAddresses))
		}
	}
}

// ScanAddressesForHighBalances quét các địa chỉ để tìm số dư lớn
func ScanAddressesForHighBalances(apiKey string, addresses []string) (map[string]string, error) {
	api := NewBscScanAPI(apiKey)

	log.Printf("Quét %d địa chỉ để tìm số dư lớn (>= %.2f BTCB)\n", len(addresses), OrganizationThreshold)

	// Map để lưu trữ địa chỉ -> số dư (chỉ cho các địa chỉ có số dư >= OrganizationThreshold)
	highBalances := make(map[string]string)

	// Lấy số dư cho từng địa chỉ
	for _, address := range addresses {
		balance, err := api.GetAddressBalance(address)
		if err != nil {
			log.Printf("Cảnh báo: Không thể lấy số dư cho địa chỉ %s: %v", address, err)
			continue
		}

		// Chuyển đổi số dư từ wei sang BTCB (định dạng dễ đọc)
		humanReadableBalance, err := convertWeiToToken(balance, BTCBDecimals)
		if err != nil {
			log.Printf("Cảnh báo: Không thể chuyển đổi số dư cho địa chỉ %s: %v", address, err)
			continue
		}

		// Chuyển đổi sang float để so sánh
		balanceFloat, _ := strconv.ParseFloat(humanReadableBalance, 64)

		// Cập nhật danh sách địa chỉ có số dư lớn
		UpdateHighBalanceAddresses(address, balanceFloat)

		// Tính phần trăm nắm giữ
		percentHold := CalculateHoldingPercentage(balanceFloat)

		// Chỉ lưu các địa chỉ có số dư >= OrganizationThreshold
		if balanceFloat >= OrganizationThreshold {
			highBalances[address] = humanReadableBalance
			usdValue := ConvertBTCBToUSD(balanceFloat)
			log.Printf("Tìm thấy địa chỉ có số dư lớn: %s với %.8f BTCB (%.2f USD) - %.6f%% tổng cung\n",
				address, balanceFloat, usdValue, percentHold)
		}
	}

	return highBalances, nil
}
