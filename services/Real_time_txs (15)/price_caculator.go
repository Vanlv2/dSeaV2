package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// TokenPriceCache lưu trữ giá token và thời gian cập nhật
type TokenPriceCache struct {
	Price      float64
	UpdateTime time.Time
}

// TokenInfo chứa thông tin về token
type TokenInfo struct {
	Symbol      string
	Decimals    int
	CoinGeckoID string // ID của token trên CoinGecko
}

// Mapping từ symbol token sang thông tin token
var tokenInfoMap = map[string]TokenInfo{
	"WBTC": {
		Symbol:      "WBTC",
		Decimals:    8,
		CoinGeckoID: "wrapped-bitcoin",
	},
	"WBNB": {
		Symbol:      "WBNB",
		Decimals:    18,
		CoinGeckoID: "binancecoin",
	},
	// Thêm các token khác nếu cần
}

// Cache lưu trữ giá token
var (
	tokenPriceCache = make(map[string]TokenPriceCache)
	cacheMutex      = &sync.RWMutex{}
	cacheExpiration = 4 * time.Minute // Thời gian hết hạn cache (4 phút)
)

// CalculateUSDValue tính giá trị USD của một lượng token
func CalculateUSDValue(tokenSymbol string, amountStr string) (float64, error) {
	// Lấy thông tin token
	tokenInfo, exists := tokenInfoMap[tokenSymbol]
	if !exists {
		return 0, fmt.Errorf("không tìm thấy thông tin cho token %s", tokenSymbol)
	}

	// Chuyển đổi amount từ string sang big.Int
	amount := new(big.Int)
	amount, ok := amount.SetString(amountStr, 10)
	if !ok {
		return 0, fmt.Errorf("không thể chuyển đổi số lượng token: %s", amountStr)
	}

	// Lấy giá token từ cache hoặc API
	tokenPrice, err := getTokenPriceWithCache(tokenInfo.CoinGeckoID)
	if err != nil {
		return 0, fmt.Errorf("không thể lấy giá token: %v", err)
	}

	// Tính toán giá trị USD
	// Chuyển đổi amount từ wei sang đơn vị token
	amountFloat := new(big.Float).SetInt(amount)
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(tokenInfo.Decimals)), nil))
	amountInToken := new(big.Float).Quo(amountFloat, divisor)

	// Nhân với giá token để có giá trị USD
	usdValue := new(big.Float).Mul(amountInToken, big.NewFloat(tokenPrice))

	// Chuyển đổi kết quả sang float64
	usdValueFloat, _ := usdValue.Float64()

	return usdValueFloat, nil
}

// getTokenPriceWithCache lấy giá token từ cache hoặc API nếu cache hết hạn
func getTokenPriceWithCache(coinGeckoID string) (float64, error) {
	cacheMutex.RLock()
	cachedData, exists := tokenPriceCache[coinGeckoID]
	cacheMutex.RUnlock()

	// Kiểm tra xem cache có tồn tại và còn hiệu lực không
	if exists && time.Since(cachedData.UpdateTime) < cacheExpiration {
		return cachedData.Price, nil
	}

	// Cache không tồn tại hoặc đã hết hạn, gọi API
	price, err := getTokenPrice(coinGeckoID)
	if err != nil {
		// Nếu có lỗi nhưng cache tồn tại, sử dụng giá trị cache cũ
		if exists {
			fmt.Printf("Cảnh báo: Không thể cập nhật giá token %s, sử dụng giá cũ: %v\n", 
				coinGeckoID, err)
			return cachedData.Price, nil
		}
		return 0, err
	}

	// Cập nhật cache
	cacheMutex.Lock()
	tokenPriceCache[coinGeckoID] = TokenPriceCache{
		Price:      price,
		UpdateTime: time.Now(),
	}
	cacheMutex.Unlock()

	return price, nil
}

// getTokenPrice lấy giá token từ CoinGecko API
func getTokenPrice(coinGeckoID string) (float64, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", coinGeckoID)
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API trả về mã lỗi: %d", resp.StatusCode)
	}
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	
	var priceResp map[string]map[string]float64
	if err := json.Unmarshal(body, &priceResp); err != nil {
		return 0, err
	}
	
	if price, ok := priceResp[coinGeckoID]["usd"]; ok {
		return price, nil
	}
	
	return 0, fmt.Errorf("không tìm thấy giá USD cho token %s", coinGeckoID)
}

// EnrichTransactionWithUSDValue thêm thông tin giá trị USD vào dữ liệu giao dịch
func EnrichTransactionWithUSDValue(txData map[string]interface{}) error {
	tokenSymbol, ok := txData["tokenSymbol"].(string)
	if !ok {
		return fmt.Errorf("không tìm thấy tokenSymbol trong dữ liệu giao dịch")
	}
	
	amountStr, ok := txData["amount"].(string)
	if !ok {
		return fmt.Errorf("không tìm thấy amount trong dữ liệu giao dịch")
	}
	
	usdValue, err := CalculateUSDValue(tokenSymbol, amountStr)
	if err != nil {
		return err
	}
	
	// Thêm giá trị USD vào dữ liệu giao dịch
	txData["usd_value"] = usdValue
	
	return nil
}

// InitTokenPrices khởi tạo giá token cho tất cả các token đã biết
func InitTokenPrices() {
	fmt.Println("Đang khởi tạo giá token...")
	
	for _, tokenInfo := range tokenInfoMap {
		_, err := getTokenPriceWithCache(tokenInfo.CoinGeckoID)
		if err != nil {
			fmt.Printf("Không thể khởi tạo giá cho token %s: %v\n", 
				tokenInfo.Symbol, err)
		} else {
			fmt.Printf("Đã khởi tạo giá cho token %s thành công\n", 
				tokenInfo.Symbol)
		}
		
		// Đợi một chút giữa các lần gọi API để tránh rate limit
		time.Sleep(500 * time.Millisecond)
	}
}
