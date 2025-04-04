package entities

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Cấu trúc dữ liệu cho thông tin giá
type PriceData struct {
	Price      float64
	LastUpdate time.Time
}

// Cấu trúc dữ liệu cho thông tin tổng cung
type SupplyData struct {
	Supply     float64
	LastUpdate time.Time
}

// Cấu trúc cache để lưu trữ giá và tổng cung
type PriceCache struct {
	mutex             sync.RWMutex
	currentPrices     map[string]PriceData
	circulatingSupply map[string]SupplyData
	cacheDuration     time.Duration
}

// Biến global để lưu trữ cache
var priceCache PriceCache

func init() {
	priceCache = PriceCache{
		currentPrices:     make(map[string]PriceData),
		circulatingSupply: make(map[string]SupplyData),
		cacheDuration:     4 * time.Minute,
	}
}

// Map các token phổ biến
var tokenIdMap = map[string]string{
	"WBTC":  "wrapped-bitcoin",
	"BTC":   "bitcoin",
	"ETH":   "ethereum",
	"BTCB":  "bitcoin-bep2",
	"USDT":  "tether",
	"USDC":  "usd-coin",
	"BNB":   "binancecoin",
	"BUSD":  "binance-usd",
	"DAI":   "dai",
	"CAKE":  "pancakeswap-token",
	"XRP":   "ripple",
	"ADA":   "cardano",
	"DOT":   "polkadot",
	"UNI":   "uniswap",
	"LINK":  "chainlink",
	"SOL":   "solana",
	"DOGE":  "dogecoin",
	"SHIB":  "shiba-inu",
	"MATIC": "matic-network",
	"AVAX":  "avalanche-2",
}

// Hàm lấy tokenId cho CoinGecko API
func getTokenIdForCoinGecko(symbol string) string {
	// Chuyển đổi symbol thành chữ thường
	tokenId, exists := tokenIdMap[symbol]
	if exists {
		return tokenId
	}
	// Nếu không tìm thấy, trả về symbol ban đầu (chữ thường)
	return symbol
}

// Hàm lấy giá hiện tại của token
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

// Hàm lấy tổng cung lưu hành của token
func getCirculatingSupply(tokenId string) (float64, error) {
	priceCache.mutex.RLock()
	cachedData, exists := priceCache.circulatingSupply[tokenId]
	priceCache.mutex.RUnlock()

	// Kiểm tra xem dữ liệu có trong cache và còn hiệu lực không
	if exists && time.Since(cachedData.LastUpdate) < priceCache.cacheDuration {
		return cachedData.Supply, nil
	}

	// Nếu không có trong cache hoặc đã hết hạn, gọi API
	url := fmt.Sprintf(
		"https://api.coingecko.com/api/v3/coins/%s",
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
	var coinData struct {
		MarketData struct {
			CirculatingSupply float64 `json:"circulating_supply"`
		} `json:"market_data"`
	}

	if err := json.Unmarshal(body, &coinData); err != nil {
		return 0, fmt.Errorf("lỗi khi parse JSON: %w", err)
	}

	supply := coinData.MarketData.CirculatingSupply

	// Cập nhật cache
	priceCache.mutex.Lock()
	if priceCache.circulatingSupply == nil {
		priceCache.circulatingSupply = make(map[string]SupplyData)
	}
	priceCache.circulatingSupply[tokenId] = SupplyData{
		Supply:     supply,
		LastUpdate: time.Now(),
	}
	priceCache.mutex.Unlock()

	return supply, nil
}
