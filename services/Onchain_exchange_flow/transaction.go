package Onchain_exchange_flow

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// --- Cấu hình API ---
const (
	// URL cơ sở của API BscScan
	bscscanBaseURL = "https://api.bscscan.com/api"
)

// --- Cấu trúc dữ liệu cho API BscScan ---
type BscScanResponse struct {
	Status  string          `json:"status"`  // Trạng thái từ API ("1" là thành công, "0" là lỗi)
	Message string          `json:"message"` // Thông điệp từ API ("OK" hoặc mô tả lỗi)
	Result  json.RawMessage `json:"result"`  // Dữ liệu kết quả thực tế
}

type TokenTransfer struct {
	BlockNumber       string `json:"blockNumber"`       // Số khối chứa giao dịch
	TimeStamp         string `json:"timeStamp"`         // Dấu thời gian Unix (dạng string)
	Hash              string `json:"hash"`              // Mã hash của giao dịch blockchain
	Nonce             string `json:"nonce"`             // Nonce của người gửi
	BlockHash         string `json:"blockHash"`         // Mã hash của khối
	From              string `json:"from"`              // Địa chỉ ví người gửi
	ContractAddress   string `json:"contractAddress"`   // Địa chỉ contract của token
	To                string `json:"to"`                // Địa chỉ ví người nhận
	Value             string `json:"value"`             // Số lượng token ở đơn vị nhỏ nhất (ví dụ: wei), dạng string
	TokenName         string `json:"tokenName"`         // Tên đầy đủ của token
	TokenSymbol       string `json:"tokenSymbol"`       // Ký hiệu của token (ví dụ: USDT)
	TokenDecimal      string `json:"tokenDecimal"`      // Số chữ số thập phân của token (dùng để chuyển đổi `Value`)
	TransactionIndex  string `json:"transactionIndex"`  // Vị trí giao dịch trong khối
	Gas               string `json:"gas"`               // Giới hạn gas
	GasPrice          string `json:"gasPrice"`          // Giá gas
	GasUsed           string `json:"gasUsed"`           // Lượng gas đã sử dụng
	CumulativeGasUsed string `json:"cumulativeGasUsed"` // Tổng gas đã sử dụng trong khối đến giao dịch này
	Input             string `json:"input"`             // Dữ liệu đầu vào của giao dịch
	Confirmations     string `json:"confirmations"`     // Số lượng khối xác nhận sau giao dịch này
}

// Cấu trúc dữ liệu đầu ra cho giao dịch đã xử lý
type ProcessedTransaction struct {
	Timestamp       int64  `json:"timestamp"`       // Unix timestamp (integer)
	TransactionType string `json:"transactionType"` // "deposit", "withdrawal", hoặc "transfer"
	AssetAmount     string `json:"assetAmount"`     // Số lượng token đã chuyển đổi
	UsdValue        string `json:"usdValue"`        // Giá trị USD tại thời điểm giao dịch
	FromAddress     string `json:"fromAddress"`     // Địa chỉ người gửi
	ToAddress       string `json:"toAddress"`       // Địa chỉ người nhận
	TokenSymbol     string `json:"tokenSymbol"`     // Ký hiệu token (ví dụ: WBTC)
	TxHash          string `json:"txHash"`          // Mã hash của giao dịch blockchain
	WalletAddress   string `json:"walletAddress"`   // Địa chỉ ví đang phân tích
}

// TransactionStats chứa thống kê về giao dịch
type TransactionStats struct {
	TotalTransactions int
	DepositCount      int
	WithdrawalCount   int
	TransferCount     int
	Transactions      []ProcessedTransaction
}

// weiToDecimal chuyển đổi giá trị token từ đơn vị nhỏ nhất sang dạng thập phân chuẩn
func weiToDecimal(weiValue string, decimals string) (string, error) {
	// Chuyển đổi giá trị wei (string) sang kiểu big.Int
	wei := new(big.Int)
	_, ok := wei.SetString(weiValue, 10)
	if !ok {
		return "", fmt.Errorf("không thể parse giá trị wei (số lượng token thô): '%s'", weiValue)
	}

	// Chuyển đổi số chữ số thập phân (string) sang integer
	dec, err := strconv.Atoi(decimals)
	if err != nil {
		return "", fmt.Errorf("không thể parse số chữ số thập phân của token: '%s'", decimals)
	}
	if dec < 0 {
		return "", fmt.Errorf("số chữ số thập phân không hợp lệ: %d", dec)
	}

	// Tạo số chia (10 mũ `decimals`) dùng big.Int
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(dec)), nil)

	// Để thực hiện phép chia và có kết quả thập phân, chuyển sang big.Float
	weiFloat := new(big.Float).SetInt(wei)
	divisorFloat := new(big.Float).SetInt(divisor)

	// Thực hiện phép chia: amount = wei / (10^decimals)
	resultFloat := new(big.Float).SetPrec(256).Quo(weiFloat, divisorFloat)

	// Định dạng kết quả big.Float thành chuỗi string
	return resultFloat.Text('f', -1), nil
}

// --- Hàm lấy lịch sử giao dịch token từ BscScan API ---
func getTokenTransfers(apiKey, walletAddr, contractAddr string) (TransactionStats, error) {
	var stats TransactionStats

	// Xây dựng URL cho yêu cầu API
	params := url.Values{}
	params.Add("module", "account")
	params.Add("action", "tokentx")
	params.Add("contractaddress", contractAddr)
	params.Add("address", walletAddr)
	params.Add("page", "1")
	params.Add("offset", "10000")
	params.Add("startblock", "0")
	params.Add("endblock", "99999999")
	params.Add("sort", "desc")
	params.Add("apikey", apiKey)

	// Tạo URL đầy đủ
	fullURL := fmt.Sprintf("%s?%s", bscscanBaseURL, params.Encode())

	// Thực hiện yêu cầu HTTP GET
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Get(fullURL)
	if err != nil {
		return stats, fmt.Errorf("lỗi khi thực hiện yêu cầu HTTP GET tới BscScan: %w", err)
	}
	defer resp.Body.Close()

	// Kiểm tra Status Code của HTTP Response
	if resp.StatusCode != http.StatusOK {
		// Cố gắng đọc body lỗi để hiển thị thông tin hữu ích hơn
		_, _ = io.ReadAll(resp.Body)
		return stats, fmt.Errorf("API BscScan trả về HTTP status code không thành công: %d", resp.StatusCode)
	}

	// Đọc nội dung (body) của response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return stats, fmt.Errorf("lỗi khi đọc nội dung response từ BscScan: %w", err)
	}

	// Parse JSON tổng quát để kiểm tra Status và Message từ API
	var apiResp BscScanResponse
	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return stats, fmt.Errorf("lỗi khi parse cấu trúc JSON response chính từ BscScan: %w", err)
	}

	// Kiểm tra Status và Message trong JSON response từ BscScan
	if apiResp.Status != "1" {
		var errorResult string
		if json.Unmarshal(apiResp.Result, &errorResult) == nil {
			return stats, fmt.Errorf("API BscScan báo lỗi: Status=%s, Message='%s'", apiResp.Status, apiResp.Message)
		}
		return stats, fmt.Errorf("API BscScan báo lỗi: Status=%s, Message='%s'", apiResp.Status, apiResp.Message)
	}

	// Parse danh sách các giao dịch token từ trường 'Result'
	var transfers []TokenTransfer
	err = json.Unmarshal(apiResp.Result, &transfers)
	if err != nil {
		return stats, fmt.Errorf("lỗi khi parse mảng giao dịch token từ BscScan Result: %w", err)
	}

	// Xử lý từng giao dịch và chuyển đổi sang định dạng mong muốn
	processedTxs := make([]ProcessedTransaction, 0, len(transfers))
	for _, tx := range transfers {
		// Chuyển đổi timestamp từ string sang int64
		ts, err := strconv.ParseInt(tx.TimeStamp, 10, 64)
		if err != nil {
			continue
		}

		// Chuyển đổi số lượng token từ đơn vị nhỏ nhất sang dạng thập phân đọc được
		amount, err := weiToDecimal(tx.Value, tx.TokenDecimal)
		if err != nil {
			continue
		}

		// Xác định loại giao dịch (deposit, withdrawal, transfer)
		var txType string
		fromAddr := tx.From
		toAddr := tx.To

		// So sánh với địa chỉ ví đang xem xét
		if toAddr == walletAddr {
			txType = "deposit"
			stats.DepositCount++
		} else if fromAddr == walletAddr {
			txType = "withdrawal"
			stats.WithdrawalCount++
		} else {
			txType = "transfer"
			stats.TransferCount++
		}

		// Lấy token ID cho CoinGecko
		tokenId := getTokenIdForCoinGecko(tx.TokenSymbol)

		// Tính giá trị USD
		usdValue, _ := calculateUSDValue(amount, ts, tokenId)

		// Tạo đối tượng ProcessedTransaction với dữ liệu đã xử lý
		processedTxs = append(processedTxs, ProcessedTransaction{
			Timestamp:       ts,
			TransactionType: txType,
			AssetAmount:     amount,
			UsdValue:        usdValue,
			FromAddress:     fromAddr,
			ToAddress:       toAddr,
			TokenSymbol:     tx.TokenSymbol,
			TxHash:          tx.Hash,
			WalletAddress:   walletAddr,
		})
	}

	stats.TotalTransactions = len(processedTxs)
	stats.Transactions = processedTxs
	return stats, nil
}
