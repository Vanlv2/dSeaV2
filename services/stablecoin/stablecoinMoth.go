package stablecoin

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"
)

type StablecoinFlowMonth struct {
	Incoming       float64
	Outgoing       float64
	CurrentBalance float64
	MonthStartTime time.Time // Thay WeekStartTime thành MonthStartTime
}

var flowsMonth = make(map[string]StablecoinFlowMonth)
var lastLogTimeMonth time.Time

func CalculateFlowMonth(tx map[string]interface{}) (map[string]FlowData, error) {
	config := ConfigStablecoin{
		Stablecoins: []StablecoinInfo{
			{Address: "0x55d398326f99059fF775485246999027B3197955", Decimals: 18, Name: "USDT", FullName: "TetherUS"},
			{Address: "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", Decimals: 18, Name: "USDC", FullName: "USD Coin"},
			{Address: "0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56", Decimals: 18, Name: "BUSD", FullName: "Binance USD"},
			{Address: "0x1AF3F329e8BE154074D8769D1FFa4eE058B1DBc3", Decimals: 18, Name: "DAI", FullName: "Dai"},
			{Address: "0x90C97F71E18723b0CF0dfa30ee176Ab653E89F68", Decimals: 18, Name: "FRAX", FullName: "Frax"},
		},
	}

	// Parse timestamp từ tx
	timestampStr, ok := tx["timestamp"].(string)
	if !ok || timestampStr == "" {
		return nil, fmt.Errorf("invalid or missing timestamp")
	}
	timestamp, err := time.Parse("2006-01-02 15:04:05", timestampStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %v", err)
	}

	// Khởi tạo lần đầu
	if lastLogTimeMonth.IsZero() {
		// Bắt đầu từ ngày đầu tiên của tháng
		lastLogTimeMonth = time.Date(timestamp.Year(), timestamp.Month(), 1, 0, 0, 0, 0, timestamp.Location())
		for _, sc := range config.Stablecoins {
			addr := strings.ToLower(sc.Address)
			flowsMonth[addr] = StablecoinFlowMonth{
				MonthStartTime: lastLogTimeMonth,
			}
		}
	}

	// Tạo map để lưu thông tin stablecoin
	stablecoinAddresses := make(map[string]bool)
	stablecoinDecimals := make(map[string]int)
	stablecoinNames := make(map[string]string)
	for _, sc := range config.Stablecoins {
		addr := strings.ToLower(sc.Address)
		stablecoinAddresses[addr] = true
		stablecoinDecimals[addr] = sc.Decimals
		stablecoinNames[addr] = sc.Name
	}

	// Kiểm tra và log khi sang tháng mới
	currentMonthStart := time.Date(timestamp.Year(), timestamp.Month(), 1, 0, 0, 0, 0, timestamp.Location())
	if !currentMonthStart.Equal(lastLogTimeMonth) {
		if err := logPreviousMonth(config, lastLogTimeMonth); err != nil {
			return nil, err
		}
		lastLogTimeMonth = currentMonthStart
	}

	// Xử lý transaction
	address, _ := tx["address"].(string)
	transactionType, _ := tx["transaction_type"].(string)
	amountStr, _ := tx["amount"].(string)

	flowDataMap := make(map[string]FlowData)
	addrLower := strings.ToLower(address)
	if stablecoinAddresses[addrLower] && (transactionType == "Deposit" || transactionType == "Withdrawal") && amountStr != "" {
		amountBig, ok := new(big.Int).SetString(amountStr, 10)
		if !ok {
			return flowDataMap, fmt.Errorf("failed to parse amount: %s", amountStr)
		}
		decimals := stablecoinDecimals[addrLower]
		amountFloat := new(big.Float).SetInt(amountBig)
		divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
		amountFloat.Quo(amountFloat, divisor)
		amount, _ := amountFloat.Float64()
		amountInBillions := amount / 1e9

		// Cập nhật flow
		scFlow := flowsMonth[addrLower]
		if transactionType == "Deposit" {
			scFlow.Incoming += amountInBillions
		} else {
			scFlow.Outgoing += amountInBillions
		}
		flowsMonth[addrLower] = scFlow
	}

	// Tạo FlowData cho tất cả stablecoin
	for _, sc := range config.Stablecoins {
		addr := strings.ToLower(sc.Address)
		scFlow := flowsMonth[addr]
		flowDataMap[sc.Name] = FlowData{
			NameCoin:  sc.FullName,
			Symbol:    sc.Name,
			StartTime: lastLogTimeMonth.Format("2006-01-02 00:00:00"),
			Incoming:  scFlow.Incoming,
			Outgoing:  scFlow.Outgoing,
			NetFlow:   scFlow.Incoming - scFlow.Outgoing,
			Balance:   scFlow.CurrentBalance + (scFlow.Incoming - scFlow.Outgoing),
			Duration:  "1m", // Thay đổi Duration thành "1m" để biểu thị tháng
		}
	}

	return flowDataMap, nil
}

func logPreviousMonth(config ConfigStablecoin, monthStart time.Time) error {
	logFile, err := os.OpenFile("flow_data.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer logFile.Close()
	logger := log.New(logFile, "", log.LstdFlags)

	for _, sc := range config.Stablecoins {
		addr := strings.ToLower(sc.Address)
		scFlow := flowsMonth[addr]
		flowData := FlowData{
			NameCoin:  sc.FullName,
			Symbol:    sc.Name,
			StartTime: monthStart.Format("2006-01-02 00:00:00"),
			Incoming:  scFlow.Incoming,
			Outgoing:  scFlow.Outgoing,
			NetFlow:   scFlow.Incoming - scFlow.Outgoing,
			Balance:   scFlow.CurrentBalance + (scFlow.Incoming - scFlow.Outgoing),
			Duration:  "1m", // Thay đổi Duration thành "1m"
		}
		flowDataJSON, err := json.MarshalIndent(flowData, "", "  ")
		if err != nil {
			return fmt.Errorf("unable to convert flow data to JSON: %v", err)
		}
		logger.Printf("Monthly FlowData for %s: %s", sc.Name, string(flowDataJSON))

		// Reset cho tháng mới
		flowsMonth[addr] = StablecoinFlowMonth{
			CurrentBalance: flowData.Balance,
			MonthStartTime: monthStart.AddDate(0, 1, 0),
		}
	}
	return nil
}
