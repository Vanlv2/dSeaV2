package Onchain_exchange_flow

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Cấu trúc dữ liệu cho API CoinGecko
type CoinGeckoHistoricalPrice struct {
	Prices [][]float64 `json:"prices"` // Mảng các cặp [timestamp, price]
}

// Cấu trúc dữ liệu cho cache giá
type PriceData struct {
	Price      float64   // Giá token
	LastUpdate time.Time // Thời điểm cập nhật giá gần nhất
}

// Cache để lưu trữ giá token
type PriceCache struct {
	mutex sync.RWMutex
	// Map với key là "tokenId_date" (vd: "bitcoin_2023-01-01") và value là giá USD
	dailyPrices map[string]float64
	// Map với key là tokenId và value là thông tin giá hiện tại
	currentPrices map[string]PriceData
	// Thời gian cache (4 phút)
	cacheDuration time.Duration
}

// Khởi tạo cache
var priceCache = PriceCache{
	dailyPrices:   make(map[string]float64),
	currentPrices: make(map[string]PriceData),
	cacheDuration: 4 * time.Minute,
}

// Hàm lấy giá hiện tại của token từ cache hoặc API
func getCurrentPrice(tokenId string) (float64, error) {
	priceCache.mutex.RLock()
	cachedData, exists := priceCache.currentPrices[tokenId]
	priceCache.mutex.RUnlock()

	// Kiểm tra xem giá có trong cache và còn hiệu lực không
	if exists && time.Since(cachedData.LastUpdate) < priceCache.cacheDuration {
		return cachedData.Price, nil
	}

	// Nếu không có trong cache hoặc đã hết hạn, gọi API
	url := fmt.Sprintf(
		"https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd",
		tokenId,
	)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("lỗi khi gọi CoinGecko API: %w", err)
	}
	defer resp.Body.Close()

	// Kiểm tra status code
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("CoinGecko API trả về status code không thành công: %d", resp.StatusCode)
	}

	// Đọc và parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("lỗi khi đọc response body: %w", err)
	}

	// Parse JSON response
	var priceResponse map[string]map[string]float64
	if err := json.Unmarshal(body, &priceResponse); err != nil {
		return 0, fmt.Errorf("lỗi khi parse JSON: %w", err)
	}

	// Kiểm tra xem có dữ liệu giá không
	priceData, exists := priceResponse[tokenId]
	if !exists {
		return 0, fmt.Errorf("không tìm thấy dữ liệu giá cho token %s", tokenId)
	}

	price, exists := priceData["usd"]
	if !exists {
		return 0, fmt.Errorf("không tìm thấy giá USD cho token %s", tokenId)
	}

	// Cập nhật cache
	priceCache.mutex.Lock()
	priceCache.currentPrices[tokenId] = PriceData{
		Price:      price,
		LastUpdate: time.Now(),
	}
	priceCache.mutex.Unlock()

	return price, nil
}

// Hàm lấy giá lịch sử của token từ CoinGecko API
func getHistoricalPrice(tokenId string, timestamp int64) (float64, error) {
	// Kiểm tra xem timestamp có nằm trong tương lai không
	currentTime := time.Now().Unix()
	if timestamp > currentTime {
		// Nếu là thời điểm trong tương lai, sử dụng giá hiện tại
		return getCurrentPrice(tokenId)
	}

	// Kiểm tra xem có phải là thời điểm hiện tại không (trong vòng 1 giờ)
	if currentTime-timestamp < 3600 {
		// Nếu là thời điểm gần đây, sử dụng giá hiện tại
		return getCurrentPrice(tokenId)
	}

	// Chuyển timestamp thành ngày để làm key cache
	date := time.Unix(timestamp, 0).Format("2006-01-02")
	cacheKey := fmt.Sprintf("%s_%s", tokenId, date)

	// Kiểm tra cache trước
	priceCache.mutex.RLock()
	if price, exists := priceCache.dailyPrices[cacheKey]; exists {
		priceCache.mutex.RUnlock()
		return price, nil
	}
	priceCache.mutex.RUnlock()

	// Tính toán khoảng thời gian 1 ngày
	startTime := time.Unix(timestamp, 0).Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)

	// Chuyển đổi thành Unix timestamp (milliseconds)
	startUnix := startTime.Unix() * 1000
	endUnix := endTime.Unix() * 1000

	// Xây dựng URL cho CoinGecko API
	url := fmt.Sprintf(
		"https://api.coingecko.com/api/v3/coins/%s/market_chart/range?vs_currency=usd&from=%d&to=%d",
		tokenId, startUnix/1000, endUnix/1000,
	)

	// Thực hiện HTTP request với retry
	client := &http.Client{Timeout: 10 * time.Second}
	var resp *http.Response
	var err error

	// Thêm cơ chế retry
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		resp, err = client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		// Đợi một chút trước khi thử lại
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return 0, fmt.Errorf("lỗi khi gọi CoinGecko API sau %d lần thử: %w", maxRetries, err)
	}
	defer resp.Body.Close()

	// Kiểm tra status code
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("CoinGecko API trả về status code không thành công: %d", resp.StatusCode)
	}

	// Đọc và parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("lỗi khi đọc response body: %w", err)
	}

	var priceData CoinGeckoHistoricalPrice
	if err := json.Unmarshal(body, &priceData); err != nil {
		return 0, fmt.Errorf("lỗi khi parse JSON: %w", err)
	}

	// Kiểm tra xem có dữ liệu giá không
	if len(priceData.Prices) == 0 {
		return 0, fmt.Errorf("không có dữ liệu giá cho %s vào ngày %s", tokenId, date)
	}

	// Tìm giá gần nhất với timestamp
	var closestPrice float64
	var minDiff int64 = 9223372036854775807 // Max int64

	for _, pricePoint := range priceData.Prices {
		if len(pricePoint) < 2 {
			continue
		}

		// CoinGecko trả về timestamp dưới dạng milliseconds
		priceTimestamp := int64(pricePoint[0]) / 1000
		diff := abs(priceTimestamp - timestamp)

		if diff < minDiff {
			minDiff = diff
			closestPrice = pricePoint[1]
		}
	}

	// Lưu vào cache
	priceCache.mutex.Lock()
	priceCache.dailyPrices[cacheKey] = closestPrice
	priceCache.mutex.Unlock()

	return closestPrice, nil
}

// Hàm tính giá trị tuyệt đối của int64
func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// Hàm tính giá trị USD của một lượng token
func calculateUSDValue(amount string, timestamp int64, tokenId string) (string, error) {
	// Parse số lượng token
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return "", fmt.Errorf("lỗi khi parse số lượng token: %w", err)
	}

	// Lấy giá USD hiện tại (thay vì giá lịch sử)
	price, err := getCurrentPrice(tokenId)
	if err != nil {
		return "N/A", err
	}

	// Tính giá trị USD
	usdValue := amountFloat * price

	// Định dạng kết quả với 2 chữ số thập phân
	return fmt.Sprintf("%.2f", usdValue), nil
}

// Hàm map từ token symbol sang token ID của CoinGecko
func getTokenIdForCoinGecko(symbol string) string {
	// Map các token phổ biến
	tokenMap := map[string]string{
		"WBTC": "wrapped-bitcoin",
		"BTC":  "bitcoin",
		"ETH":  "ethereum",
		"USDT": "tether",
		"USDC": "usd-coin",
		"BNB":  "binancecoin",
	}

	if id, exists := tokenMap[symbol]; exists {
		return id
	}

	// Trả về lowercase của symbol nếu không tìm thấy mapping
	return symbol
}
