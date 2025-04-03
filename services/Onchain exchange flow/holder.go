package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"
)

// --- Cấu hình API ---
const (
	// Địa chỉ cơ sở của API Covalent
	covalentBaseURL = "https://api.covalenthq.com/v1"

	// Chain ID cho Binance Smart Chain Mainnet
	bscChainID = "56"
)

// --- Cấu trúc dữ liệu cho API Covalent ---
type CovalentTokenHoldersResponse struct {
	Data         CovalentData `json:"data"`
	Error        bool         `json:"error"`
	ErrorMessage string       `json:"error_message"`
	ErrorCode    *int         `json:"error_code"` // Dùng con trỏ vì có thể là null
}

type CovalentData struct {
	UpdatedAt  time.Time    `json:"updated_at"`
	Items      []HolderItem `json:"items"`
	Pagination *Pagination  `json:"pagination"` // Dùng con trỏ vì có thể là null
}

type HolderItem struct {
	ContractDecimals     int    `json:"contract_decimals"`      // Số thập phân của token
	ContractName         string `json:"contract_name"`          // Tên token
	ContractTickerSymbol string `json:"contract_ticker_symbol"` // Ký hiệu token
	ContractAddress      string `json:"contract_address"`       // Địa chỉ contract token
	LogoURL              string `json:"logo_url"`               // URL logo token
	Address              string `json:"address"`                // Địa chỉ ví holder
	Balance              string `json:"balance"`                // Số dư (dạng chuỗi, đơn vị nhỏ nhất)
}

type Pagination struct {
	HasMore    bool `json:"has_more"`
	PageIndex  int  `json:"page_number"`
	PageSize   int  `json:"page_size"`
	TotalCount *int `json:"total_count"` // Dùng con trỏ vì có thể là null
}

// TokenInfo chứa thông tin cơ bản về token
type TokenInfo struct {
	Name     string
	Symbol   string
	Decimals int
}

// HolderInfo chứa thông tin về một holder
type HolderInfo struct {
	Address       string
	Balance       string
	ReadableValue *big.Float
}

// --- Hàm chuyển đổi số dư từ đơn vị nhỏ nhất (dạng chuỗi) sang big.Float ---
func convertBalance(balanceStr string, decimals int) (*big.Float, error) {
	balanceWei := new(big.Int)
	_, success := balanceWei.SetString(balanceStr, 10)
	if !success {
		return nil, fmt.Errorf("không thể chuyển đổi chuỗi balance '%s' thành big.Int", balanceStr)
	}

	// Tạo giá trị 10^decimals
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))

	// Chuyển balanceWei sang big.Float
	balanceBigFloat := new(big.Float).SetInt(balanceWei)

	// Thực hiện phép chia
	result := new(big.Float).Quo(balanceBigFloat, divisor)
	return result, nil
}

// --- Hàm lấy top token holders từ Covalent API ---
func getBSCTopTokenHolders(apiKey string, tokenAddress string, pageSize int) (TokenInfo, []HolderInfo, error) {
	var tokenInfo TokenInfo
	var holders []HolderInfo

	if apiKey == "" {
		return tokenInfo, holders, fmt.Errorf("API Key của Covalent không được cung cấp")
	}

	// Xây dựng URL request
	url := fmt.Sprintf("%s/%s/tokens/%s/token_holders/?key=%s&page-size=%d&page-number=1",
		covalentBaseURL, bscChainID, tokenAddress, apiKey, pageSize)

	// Tạo HTTP client với timeout
	client := http.Client{
		Timeout: 60 * time.Second,
	}

	// Thực hiện GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return tokenInfo, holders, fmt.Errorf("lỗi khi tạo request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return tokenInfo, holders, fmt.Errorf("lỗi khi thực hiện request: %w", err)
	}
	defer resp.Body.Close()

	// Đọc nội dung response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tokenInfo, holders, fmt.Errorf("lỗi khi đọc response body: %w", err)
	}

	// Kiểm tra HTTP status code
	if resp.StatusCode != http.StatusOK {
		// Cố gắng parse lỗi từ Covalent nếu có thể
		var errorResp CovalentTokenHoldersResponse
		_ = json.Unmarshal(body, &errorResp) // Bỏ qua lỗi unmarshal nếu đây không phải JSON lỗi chuẩn
		if errorResp.Error {
			return tokenInfo, holders, fmt.Errorf("API Covalent trả về lỗi: %d - %s (Code: %v)", resp.StatusCode, errorResp.ErrorMessage, errorResp.ErrorCode)
		}
		return tokenInfo, holders, fmt.Errorf("API trả về status không thành công: %d", resp.StatusCode)
	}

	// Unmarshal JSON data vào struct
	var holdersResp CovalentTokenHoldersResponse
	err = json.Unmarshal(body, &holdersResp)
	if err != nil {
		return tokenInfo, holders, fmt.Errorf("lỗi khi unmarshal JSON: %w", err)
	}

	// Kiểm tra lỗi logic từ Covalent (trường `error` trong JSON)
	if holdersResp.Error {
		return tokenInfo, holders, fmt.Errorf("lỗi từ API Covalent: %s (Code: %v)", holdersResp.ErrorMessage, holdersResp.ErrorCode)
	}

	if len(holdersResp.Data.Items) == 0 {
		return tokenInfo, holders, fmt.Errorf("không tìm thấy holder nào cho token này")
	}

	// Lấy thông tin token từ item đầu tiên
	firstItem := holdersResp.Data.Items[0]
	tokenInfo = TokenInfo{
		Name:     firstItem.ContractName,
		Symbol:   firstItem.ContractTickerSymbol,
		Decimals: firstItem.ContractDecimals,
	}

	// Xử lý danh sách holders
	holders = make([]HolderInfo, 0, len(holdersResp.Data.Items))
	for _, item := range holdersResp.Data.Items {
		readableBalance, _ := convertBalance(item.Balance, item.ContractDecimals)
		holders = append(holders, HolderInfo{
			Address:       item.Address,
			Balance:       item.Balance,
			ReadableValue: readableBalance,
		})
	}

	return tokenInfo, holders, nil
}
