package main

import (
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"strconv"
	"time"
)

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
			if _, exists := exchangeAddresses[address]; !exists {
				log.Printf("[PHÂN LOẠI] Địa chỉ %s được phân loại là SÀN GIAO DỊCH với %.8f BTCB (%.4f%%)", 
					address, balance, percentHold)
			}
			exchangeAddresses[address] = AddressInfo{
				Balance:    balance,
				Percentage: percentHold,
			}
		} else {
			// Địa chỉ tổ chức (100-1000 BTCB)
			if _, exists := organizationAddresses[address]; !exists {
				log.Printf("[PHÂN LOẠI] Địa chỉ %s được phân loại là TỔ CHỨC với %.8f BTCB (%.4f%%)", 
					address, balance, percentHold)
			}
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

		transactions, err := api.GetLatestTransactions(lastCheckTime)
		if err != nil {
			log.Printf("[LỖI] Không thể lấy giao dịch mới: %v", err)
			continue
		}

		if len(transactions) > 0 {
			// Cập nhật thời gian kiểm tra cuối cùng
			lastCheckTime = time.Now().Unix()

			// Thu thập các địa chỉ mới từ giao dịch
			newAddresses := make(map[string]bool)
			for _, tx := range transactions {
				newAddresses[tx.From] = true
				newAddresses[tx.To] = true
			}

			// Thêm các địa chỉ mới vào danh sách
			newCount := 0
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
					newCount++
				}
			}
			
			if newCount > 0 {
				log.Printf("[CẬP NHẬT] Đã thêm %d địa chỉ mới vào danh sách theo dõi, tổng số: %d", 
					newCount, len(allAddresses))
			}
		}
	}
}

// ScanAddressesForHighBalances quét các địa chỉ để tìm số dư lớn
func ScanAddressesForHighBalances(apiKey string, addresses []string) (map[string]string, error) {
	api := NewBscScanAPI(apiKey)

	// Map để lưu trữ địa chỉ -> số dư (chỉ cho các địa chỉ có số dư >= OrganizationThreshold)
	highBalances := make(map[string]string)
	foundCount := 0

	// Lấy số dư cho từng địa chỉ
	for _, address := range addresses {
		balance, err := api.GetAddressBalance(address)
		if err != nil {
			continue
		}

		// Chuyển đổi số dư từ wei sang BTCB (định dạng dễ đọc)
		humanReadableBalance, err := convertWeiToToken(balance, BTCBDecimals)
		if err != nil {
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
			foundCount++
			
			// Chỉ log khi tìm thấy địa chỉ có số dư lớn mới
			if _, exists := highBalanceAddresses[address]; !exists {
				log.Printf("[TÌM THẤY] Địa chỉ có số dư lớn: %s với %.8f BTCB (%.2f USD) - %.6f%% tổng cung",
					address, balanceFloat, usdValue, percentHold)
			}
		}
	}
	
	if foundCount > 0 {
		log.Printf("[QUÉT] Đã tìm thấy %d địa chỉ có số dư lớn (>= %.2f BTCB)", foundCount, OrganizationThreshold)
	}

	return highBalances, nil
}
