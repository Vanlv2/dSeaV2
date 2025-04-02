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

const CoinGeckoAPIURL = "https://api.coingecko.com/api/v3/simple/price?ids=binance-bitcoin&vs_currencies=usd"

type CoinGeckoResponse struct {
	BinanceBitcoin struct {
		USD float64 `json:"usd"`
	} `json:"binance-bitcoin"`
}

// Thời gian đợi giữa các lần quét
const ScanInterval = 2 * time.Second

// Số lượng địa chỉ quét mỗi lần
const AddressesPerScan = 100

// Lưu giá BTCB/USD để sử dụng trong chương trình
var currentBTCBPrice float64

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
	for {
		// Cập nhật giá
		price, err := GetBTCBPriceInUSD()
		if err != nil {
			log.Printf("Cảnh báo: Không thể cập nhật giá BTCB/USD: %v", err)
		} else {
			currentBTCBPrice = price
			log.Printf("Đã cập nhật giá BTCB/USD: %.2f USD", currentBTCBPrice)
		}

		// Đợi 5 phút trước khi cập nhật lại giá
		time.Sleep(5 * time.Minute)
	}
}

// ScanAddressesForHighBalancesWithUSD quét các địa chỉ để tìm số dư lớn và quy đổi sang USD
func ScanAddressesForHighBalancesWithUSD(apiKey string, addresses []string) (map[string]string, error) {
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

		humanReadableBalance, err := convertWeiToToken(balance, BTCBDecimals)
		if err != nil {
			log.Printf("Cảnh báo: Không thể chuyển đổi số dư cho địa chỉ %s: %v", address, err)
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
			log.Printf("Tìm thấy địa chỉ có số dư lớn: %s với %.8f BTCB (%.2f USD) - %.6f%% tổng cung\n",
				address, balanceFloat, usdValue, percentHold)
		}
	}

	return highBalances, nil
}

// handleValueAddress là hàm chính để quét và theo dõi các địa chỉ BTCB
func handleValueAddress() {
	apiKey := "TZR6PYQJPSREBUJHTYWP948TXHD3MXNQ7W" // Thay thế bằng API key BscScan thực tế của bạn

	// Khởi tạo bộ sinh số ngẫu nhiên
	rand.Seed(time.Now().UnixNano())

	// Lấy giá BTCB/USD ban đầu
	var err error
	currentBTCBPrice, err = GetBTCBPriceInUSD()
	if err != nil {
		log.Printf("Cảnh báo: Không thể lấy giá BTCB/USD ban đầu: %v", err)
		currentBTCBPrice = 0 // Giá mặc định
	} else {
		log.Printf("Giá BTCB/USD ban đầu: %.2f USD", currentBTCBPrice)
	}

	// Bắt đầu goroutine để cập nhật giá BTCB/USD định kỳ
	go UpdatePriceRegularly()

	// Thu thập tất cả các địa chỉ ban đầu
	allAddresses, err = CollectAllAddresses(apiKey)
	if err != nil {
		log.Fatalf("Lỗi khi thu thập địa chỉ: %v", err)
	}

	// Bắt đầu một goroutine để theo dõi các giao dịch mới
	go MonitorNewTransactions(apiKey)

	// Chạy vòng lặp quét chính
	for {
		log.Println("Bắt đầu quét các địa chỉ BTCB để tìm số dư lớn...")

		var addressesToScan []string

		// Ưu tiên quét lại các địa chỉ đã biết có số dư lớn
		highBalanceAddrs := GetHighBalanceAddresses()
		if len(highBalanceAddrs) > 0 {
			log.Printf("Ưu tiên quét %d địa chỉ đã biết có số dư lớn", len(highBalanceAddrs))
			addressesToScan = append(addressesToScan, highBalanceAddrs...)
		}

		// Bổ sung thêm các địa chỉ ngẫu nhiên nếu cần
		remainingSlots := AddressesPerScan - len(addressesToScan)
		if remainingSlots > 0 && len(allAddresses) > 0 {
			randomAddresses := GetRandomAddresses(allAddresses, remainingSlots)
			addressesToScan = append(addressesToScan, randomAddresses...)
		}

		// Quét các địa chỉ để tìm số dư lớn và quy đổi sang USD
		highBalances, err := ScanAddressesForHighBalancesWithUSD(apiKey, addressesToScan)
		if err != nil {
			log.Printf("Lỗi khi quét số dư: %v", err)
		} else {
			log.Printf("Quét hoàn tất. Tìm thấy %d địa chỉ có số dư BTCB >= %.2f",
				len(highBalances), OrganizationThreshold)
			log.Printf("Tổng số địa chỉ có số dư lớn đã biết: %d", len(highBalanceAddresses))
			log.Printf("Số địa chỉ sàn giao dịch (>%.2f BTCB): %d", ExchangeThreshold, len(exchangeAddresses))
			log.Printf("Số địa chỉ tổ chức (%.2f-%.2f BTCB): %d",
				OrganizationThreshold, ExchangeThreshold, len(organizationAddresses))
		}

		// In ra một số địa chỉ có số dư lớn nhất (nếu có)
		if len(highBalanceAddresses) > 0 {
			log.Println("Một số địa chỉ có số dư lớn nhất:")

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

			// In ra 5 địa chỉ có số dư lớn nhất
			count := 0
			for _, item := range sortedAddresses {
				if count >= 5 {
					break
				}
				log.Printf("  %s: %.8f BTCB (%.2f USD) - %.6f%% tổng cung - %s",
					item.Address, item.Balance, item.USDValue, item.Percentage, item.Type)
				count++
			}
		}

		log.Printf("Đợi %v trước khi quét lại...", ScanInterval)
		time.Sleep(ScanInterval)
	}
}
