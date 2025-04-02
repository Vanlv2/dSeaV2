package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	BscScanAPIBaseURL = "https://api.bscscan.com/api"
	BTCBTokenAddress  = "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c" // Địa chỉ token BTCB trên BSC
)

// RateLimiter giới hạn tốc độ gọi API
type RateLimiter struct {
	mu      sync.Mutex
	lastReq time.Time
	delay   time.Duration
}

// NewRateLimiter tạo rate limiter mới
func NewRateLimiter(requestsPerSecond float64) *RateLimiter {
	return &RateLimiter{
		delay: time.Duration(float64(time.Second) / requestsPerSecond),
	}
}

// Wait chờ đến khi có thể gửi yêu cầu tiếp theo
func (r *RateLimiter) Wait() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	sleepTime := r.delay - now.Sub(r.lastReq)
	if sleepTime > 0 {
		time.Sleep(sleepTime)
	}
	r.lastReq = time.Now()
}

// TokenTransaction cấu trúc dữ liệu cho giao dịch token
type TokenTransaction struct {
	BlockNumber       string `json:"blockNumber"`
	TimeStamp         string `json:"timeStamp"`
	Hash              string `json:"hash"`
	From              string `json:"from"`
	To                string `json:"to"`
	Value             string `json:"value"`
	ContractAddress   string `json:"contractAddress"`
	TokenName         string `json:"tokenName"`
	TokenSymbol       string `json:"tokenSymbol"`
	TokenDecimal      string `json:"tokenDecimal"`
	TransactionIndex  string `json:"transactionIndex"`
	Gas               string `json:"gas"`
	GasPrice          string `json:"gasPrice"`
	GasUsed           string `json:"gasUsed"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	Input             string `json:"input"`
	Confirmations     string `json:"confirmations"`
}

// TokenTxResponse cấu trúc phản hồi cho API giao dịch token
type TokenTxResponse struct {
	Status  string             `json:"status"`
	Message string             `json:"message"`
	Result  []TokenTransaction `json:"result"`
}

// TokenBalanceResponse cấu trúc phản hồi cho API số dư token
type TokenBalanceResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

// BscScanAPI wrapper cho BscScan API
type BscScanAPI struct {
	APIKey      string
	RateLimiter *RateLimiter
}

// NewBscScanAPI tạo instance mới của BscScanAPI
func NewBscScanAPI(apiKey string) *BscScanAPI {
	return &BscScanAPI{
		APIKey:      apiKey,
		RateLimiter: NewRateLimiter(4), // Giới hạn 4 yêu cầu/giây
	}
}

// RateLimitedRequest thực hiện yêu cầu HTTP với giới hạn tốc độ
func (b *BscScanAPI) RateLimitedRequest(url string) ([]byte, error) {
	b.RateLimiter.Wait()

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi thực hiện yêu cầu HTTP: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi đọc body phản hồi: %v", err)
	}

	return body, nil
}

// GetAddressBalance lấy số dư BTCB của một địa chỉ
func (b *BscScanAPI) GetAddressBalance(address string) (string, error) {
	url := fmt.Sprintf("%s?module=account&action=tokenbalance&contractaddress=%s&address=%s&tag=latest&apikey=%s",
		BscScanAPIBaseURL, BTCBTokenAddress, address, b.APIKey)

	resp, err := b.RateLimitedRequest(url)
	if err != nil {
		return "", fmt.Errorf("lỗi khi thực hiện yêu cầu tới API BscScan: %v", err)
	}

	var response TokenBalanceResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return "", fmt.Errorf("lỗi khi giải mã phản hồi: %v", err)
	}

	if response.Status != "1" {
		log.Printf("Cảnh báo: Không thể lấy số dư cho địa chỉ %s, trả về 0\n", address)
		return "0", nil
	}

	return response.Result, nil
}

// GetRecentBTCBAddresses lấy danh sách các địa chỉ từ các giao dịch BTCB gần đây
func (b *BscScanAPI) GetRecentBTCBAddresses() ([]string, error) {
	url := fmt.Sprintf("%s?module=account&action=tokentx&contractaddress=%s&page=1&offset=1000&sort=desc&apikey=%s",
		BscScanAPIBaseURL, BTCBTokenAddress, b.APIKey)

	resp, err := b.RateLimitedRequest(url)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi lấy giao dịch token: %v", err)
	}

	var txResponse TokenTxResponse
	if err := json.Unmarshal(resp, &txResponse); err != nil {
		return nil, fmt.Errorf("lỗi khi giải mã phản hồi giao dịch: %v", err)
	}

	if txResponse.Status != "1" {
		return nil, fmt.Errorf("lỗi API: %s", txResponse.Message)
	}

	// Tạo map để loại bỏ các địa chỉ trùng lặp
	addressMap := make(map[string]bool)
	for _, tx := range txResponse.Result {
		addressMap[tx.From] = true
		addressMap[tx.To] = true
	}

	// Chuyển đổi map thành slice
	var addresses []string
	for addr := range addressMap {
		addresses = append(addresses, addr)
	}

	log.Printf("Lấy được %d địa chỉ từ các giao dịch gần đây\n", len(addresses))
	return addresses, nil
}

// GetLatestTransactions lấy các giao dịch BTCB mới nhất sau thời điểm startTime
func (b *BscScanAPI) GetLatestTransactions(startTime int64) ([]TokenTransaction, error) {
	url := fmt.Sprintf("%s?module=account&action=tokentx&contractaddress=%s&page=1&offset=5000&sort=desc&apikey=%s",
		BscScanAPIBaseURL, BTCBTokenAddress, b.APIKey)

	resp, err := b.RateLimitedRequest(url)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi lấy giao dịch token: %v", err)
	}

	var txResponse TokenTxResponse
	if err := json.Unmarshal(resp, &txResponse); err != nil {
		return nil, fmt.Errorf("lỗi khi giải mã phản hồi giao dịch: %v", err)
	}

	if txResponse.Status != "1" {
		return nil, fmt.Errorf("lỗi API: %s", txResponse.Message)
	}

	// Lọc các giao dịch sau thời gian startTime
	var recentTxs []TokenTransaction
	for _, tx := range txResponse.Result {
		txTime, err := strconv.ParseInt(tx.TimeStamp, 10, 64)
		if err == nil && txTime > startTime {
			recentTxs = append(recentTxs, tx)
		}
	}

	log.Printf("Lấy được %d giao dịch mới\n", len(recentTxs))
	return recentTxs, nil
}
