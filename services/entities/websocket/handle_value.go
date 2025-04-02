package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// GetBTCBPriceInUSD lấy tỷ giá BTCB sang USD từ CoinGecko
func GetBTCBPriceInUSD() (float64, error) {
	resp, err := http.Get(CoinGeckoAPIURL)
	if err != nil {
		return 0, fmt.Errorf("lỗi khi gọi API CoinGecko: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("lỗi khi đọc dữ liệu từ API CoinGecko: %v", err)
	}

	var priceResponse CoinGeckoResponse
	if err := json.Unmarshal(body, &priceResponse); err != nil {
		return 0, fmt.Errorf("lỗi khi giải mã dữ liệu từ API CoinGecko: %v", err)
	}

	return priceResponse.BinanceBitcoin.USD, nil
}

// ConvertBTCBToUSD chuyển đổi số dư BTCB sang USD
func ConvertBTCBToUSD(btcbAmount float64) float64 {
	return btcbAmount * currentBTCBPrice
}

// UpdatePriceRegularly cập nhật giá BTCB/USD định kỳ
func UpdatePriceRegularly() {
	wg.Add(1)
	defer wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			// Cập nhật giá
			price, err := GetBTCBPriceInUSD()
			if err != nil {
				log.Printf("[CẢNH BÁO] Không thể cập nhật giá BTCB/USD: %v", err)
			} else {
				currentBTCBPrice = price
				log.Printf("[GIÁ] BTCB/USD: %.2f USD", currentBTCBPrice)
			}
		}
	}
}

// ScanAddressesForHighBalancesWithUSD quét các địa chỉ để tìm số dư lớn và quy đổi sang USD
func ScanAddressesForHighBalancesWithUSD(apiKey string, addresses []string) (map[string]string, error) {
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

		humanReadableBalance, err := convertWeiToToken(balance, BTCBDecimals)
		if err != nil {
			continue
		}

		// Chuyển đổi sang float để so sánh
		balanceFloat, _ := strconv.ParseFloat(humanReadableBalance, 64)

		// Tính giá trị USD và phần trăm nắm giữ
		usdValue := ConvertBTCBToUSD(balanceFloat)
		percentHold := CalculateHoldingPercentage(balanceFloat)

		// Cập nhật danh sách địa chỉ có số dư lớn
		UpdateHighBalanceAddresses(address, balanceFloat)

		// Chỉ lưu các địa chỉ có số dư >= OrganizationThreshold
		if balanceFloat >= OrganizationThreshold {
			highBalances[address] = humanReadableBalance
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

// handleValueAddress là hàm chính để quét và theo dõi các địa chỉ BTCB
func handleValueAddress() {
	// Khởi tạo bộ sinh số ngẫu nhiên
	rand.Seed(time.Now().UnixNano())

	// Lấy giá BTCB/USD ban đầu
	var err error
	currentBTCBPrice, err = GetBTCBPriceInUSD()
	if err != nil {
		log.Printf("[CẢNH BÁO] Không thể lấy giá BTCB/USD ban đầu: %v", err)
		currentBTCBPrice = 0 // Giá mặc định
	} else {
		log.Printf("[GIÁ] BTCB/USD ban đầu: %.2f USD", currentBTCBPrice)
	}

	// Bắt đầu goroutine để cập nhật giá BTCB/USD định kỳ
	go UpdatePriceRegularly()

	// Thu thập tất cả các địa chỉ ban đầu
	allAddresses, err = CollectAllAddresses(apiKey)
	if err != nil {
		log.Fatalf("[LỖI] Không thể thu thập địa chỉ: %v", err)
	}
	log.Printf("[KHỞI TẠO] Đã thu thập %d địa chỉ ban đầu", len(allAddresses))

	// Bắt đầu một goroutine để theo dõi các giao dịch mới
	go MonitorNewTransactions(apiKey)

	// Thời gian đợi giữa các lần quét
	const ScanInterval = 4 * time.Second
	// Số lượng địa chỉ quét mỗi lần
	const AddressesPerScan = 100

	// Chạy vòng lặp quét chính
	log.Println("[BẮT ĐẦU] Quét các địa chỉ BTCB để tìm số dư lớn...")

	for {
		var addressesToScan []string

		// Ưu tiên quét lại các địa chỉ đã biết có số dư lớn
		highBalanceAddrs := GetHighBalanceAddresses()
		if len(highBalanceAddrs) > 0 {
			addressesToScan = append(addressesToScan, highBalanceAddrs...)
		}

		// Bổ sung thêm các địa chỉ ngẫu nhiên nếu cần
		remainingSlots := AddressesPerScan - len(addressesToScan)
		if remainingSlots > 0 && len(allAddresses) > 0 {
			randomAddresses := GetRandomAddresses(allAddresses, remainingSlots)
			addressesToScan = append(addressesToScan, randomAddresses...)
		}

		// Quét các địa chỉ để tìm số dư lớn và quy đổi sang USD
		_, err := ScanAddressesForHighBalancesWithUSD(apiKey, addressesToScan)
		if err != nil {
			log.Printf("[LỖI] Không thể quét số dư: %v", err)
		} else {
			// Chỉ in thống kê định kỳ (mỗi 10 giây)
			if time.Now().Second()%10 == 0 && time.Now().Second() < 5 {
				log.Printf("[THỐNG KÊ] Tổng số địa chỉ có số dư lớn: %d", len(highBalanceAddresses))
				log.Printf("[THỐNG KÊ] Số địa chỉ sàn giao dịch (>%.2f BTCB): %d", ExchangeThreshold, len(exchangeAddresses))
				log.Printf("[THỐNG KÊ] Số địa chỉ tổ chức (%.2f-%.2f BTCB): %d",
					OrganizationThreshold, ExchangeThreshold, len(organizationAddresses))

				// In ra một số địa chỉ có số dư lớn nhất (nếu có)
				if len(highBalanceAddresses) > 0 {
					log.Println("[TOP] Các địa chỉ có số dư lớn nhất:")

					// Sắp xếp các địa chỉ theo số dư giảm dần
					type AddressBalance struct {
						Address    string
						Balance    float64
						USDValue   float64
						Percentage float64
						Type       string
					}

					sortedAddresses := make([]AddressBalance, 0, len(highBalanceAddresses))
					for addr, info := range highBalanceAddresses {
						usdValue := ConvertBTCBToUSD(info.Balance)
						addrType := "Tổ chức"
						if info.Balance >= ExchangeThreshold {
							addrType = "Sàn giao dịch"
						}

						sortedAddresses = append(sortedAddresses, AddressBalance{
							Address:    addr,
							Balance:    info.Balance,
							USDValue:   usdValue,
							Percentage: info.Percentage,
							Type:       addrType,
						})
					}

					sort.Slice(sortedAddresses, func(i, j int) bool {
						return sortedAddresses[i].Balance > sortedAddresses[j].Balance
					})

					// Thay thế đoạn mã từ dòng 214-223 bằng đoạn này
					// In ra tất cả các địa chỉ có số dư lớn
					for _, item := range sortedAddresses {
						log.Printf("  %s: %.8f BTCB (%.2f USD) - %.6f%% tổng cung - %s",
							item.Address, item.Balance, item.USDValue, item.Percentage, item.Type)
					}
				}
			}
		}

		time.Sleep(ScanInterval)
	}
}
