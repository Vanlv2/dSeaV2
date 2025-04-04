package entities

import (
	"fmt"
	"math/big"
	"time"
)

// --- Cấu hình API ---
const (
	covalentBaseURL = "https://api.covalenthq.com/v1"
	bscChainID      = "56"
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
	ContractAddress      string `json:"contract_address"`       // Địa chỉ contract
	LogoURL              string `json:"logo_url"`               // URL logo token - Đã sửa từ bool thành string
	Address              string `json:"address"`                // Địa chỉ ví
	Balance              string `json:"balance"`                // Số dư token (dạng string)
	TotalSupply          string `json:"total_supply"`           // Tổng cung token
	BlockHeight          int    `json:"block_height"`           // Chiều cao block
}

type Pagination struct {
	HasMore    bool `json:"has_more"`
	PageNumber int  `json:"page_number"`
	PageSize   int  `json:"page_size"`
	TotalCount *int `json:"total_count"` // Dùng con trỏ vì có thể là null
}

// Hàm chuyển đổi balance từ string sang số thực
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
