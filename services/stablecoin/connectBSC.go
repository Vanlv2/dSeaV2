package stablecoin

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func listenToTransactions(client *ethclient.Client, config ConfigStablecoin) {
	// Tạo danh sách địa chỉ stablecoin để lọc
	addresses := make([]common.Address, len(config.Stablecoins))
	for i, sc := range config.Stablecoins {
		addresses[i] = common.HexToAddress(sc.Address)
	}

	// Thiết lập filter cho sự kiện Transfer
	query := ethereum.FilterQuery{
		Addresses: addresses,
		Topics:    [][]common.Hash{{common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")}}, // Transfer event
	}

	// Kênh để nhận log
	logs := make(chan types.Log)
	ctx := context.Background()

	// Đăng ký lắng nghe sự kiện
	sub, err := client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		log.Fatalf("Failed to subscribe to logs: %v", err)
	}
	defer sub.Unsubscribe()

	fmt.Println("Listening for real-time transactions...")

	// Xử lý log theo thời gian thực
	for {
		select {
		case err := <-sub.Err():
			log.Fatalf("Subscription error: %v", err)
		case vLog := <-logs:
			// Lấy timestamp từ block
			timestamp, err := getBlockTimestamp(ctx, client, vLog.BlockNumber)
			if err != nil {
				log.Printf("Failed to get timestamp: %v", err)
				continue
			}

			// Xử lý dữ liệu log
			contractAddress := vLog.Address.Hex()
			from := common.BytesToAddress(vLog.Topics[1].Bytes()[12:]).Hex()
			amount := new(big.Int).SetBytes(vLog.Data).String()
			txHash := vLog.TxHash.Hex()

			// Xác định transaction_type (giả định đơn giản)
			txType := "Deposit"
			if strings.EqualFold(from, contractAddress) {
				txType = "Withdrawal"
			}

			// Tạo tx map
			tx := map[string]interface{}{
				"timestamp":        timestamp.Format("2006-01-02 15:04:05"),
				"address":          contractAddress,
				"transaction_type": txType,
				"amount":           amount,
				"tx_hash":          txHash,
			}
			flowDataDate(tx)
			time.Sleep(5 * time.Second)
			flowDataWeek(tx)
			time.Sleep(5 * time.Second)
			flowDataMonth(tx)
		}
	}
}

func getBlockTimestamp(ctx context.Context, client *ethclient.Client, blockNumber uint64) (time.Time, error) {
	block, err := client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(block.Time()), 0), nil
}

func Stablecoin() {
	// Kết nối đến node BSC qua WebSocket
	nodeURL := "wss://bsc-mainnet.core.chainstack.com/8e6310f0dc371b60ddc0de98a4d5d1e3" // WebSocket công cộng BSC
	client, err := ethclient.Dial(nodeURL)
	if err != nil {
		log.Fatalf("Failed to connect to BSC node: %v", err)
	}
	defer client.Close()

	// Cấu hình stablecoin
	config := ConfigStablecoin{
		Stablecoins: []StablecoinInfo{
			{Address: "0x55d398326f99059fF775485246999027B3197955", Decimals: 18, Name: "USDT"},
			{Address: "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", Decimals: 18, Name: "USDC"},
			{Address: "0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56", Decimals: 18, Name: "BUSD"},
			{Address: "0x1AF3F329e8BE154074D8769D1FFa4eE058B1DBc3", Decimals: 18, Name: "DAI"},
			{Address: "0x90C97F71E18723b0CF0dfa30ee176Ab653E89F68", Decimals: 18, Name: "FRAX"},
		},
	}
	// Bắt đầu lắng nghe giao dịch real-time
	listenToTransactions(client, config)
}

func flowDataDate(tx map[string]interface{}) {
	// Tính toán flow
	flowDataMap, err := CalculateFlowDate(tx)
	if err != nil {
		log.Printf("Error calculating flow: %v", err)
		return
	}

	// In kết quả
	fmt.Printf("Processed tx %s at %s:\n", tx["tx_hash"], tx["timestamp"])
	for name, flow := range flowDataMap {
		// Chuyển đổi StartTime từ string sang int
		startTimeParsed, err := time.Parse("2006-01-02 15:04:05", flow.StartTime)
		if err != nil {
			log.Printf("Error parsing StartTime: %v", err)
			continue
		}
		startTime := startTimeParsed.Unix()
		incoming := strconv.FormatFloat(flow.Incoming, 'f', -1, 64)
		outgoing := strconv.FormatFloat(flow.Outgoing, 'f', -1, 64)
		balance := strconv.FormatFloat(flow.Balance, 'f', -1, 64)
		// fmt.Printf("  %s: %+v\n", name, flow)
		StablecoinSMCDate(startTime, incoming, outgoing, balance, name, flow.NameCoin)
	}
}
func flowDataWeek(tx map[string]interface{}) {
	// Tính toán flow
	flowDataMapWeek, err := CalculateFlowWeek(tx)
	if err != nil {
		log.Printf("Error calculating flow: %v", err)
		return
	}

	// In kết quả
	fmt.Printf("Processed tx %s at %s:\n", tx["tx_hash"], tx["timestamp"])
	for name, flow := range flowDataMapWeek {
		// Chuyển đổi StartTime từ string sang int
		startTimeParsed, err := time.Parse("2006-01-02 15:04:05", flow.StartTime)
		if err != nil {
			log.Printf("Error parsing StartTime: %v", err)
			continue
		}
		startTime := startTimeParsed.Unix()
		incoming := strconv.FormatFloat(flow.Incoming, 'f', -1, 64)
		outgoing := strconv.FormatFloat(flow.Outgoing, 'f', -1, 64)
		balance := strconv.FormatFloat(flow.Balance, 'f', -1, 64)
		// fmt.Printf("  %s: %+v\n", name, flow)
		StablecoinSMCWeek(startTime, incoming, outgoing, balance, name, flow.NameCoin)
	}
}
func flowDataMonth(tx map[string]interface{}) {
	// Tính toán flow
	flowDataMapMonth, err := CalculateFlowMonth(tx)
	if err != nil {
		log.Printf("Error calculating flow: %v", err)
		return
	}

	// In kết quả
	fmt.Printf("Processed tx %s at %s:\n", tx["tx_hash"], tx["timestamp"])
	for name, flow := range flowDataMapMonth {
		// Chuyển đổi StartTime từ string sang int
		startTimeParsed, err := time.Parse("2006-01-02 15:04:05", flow.StartTime)
		if err != nil {
			log.Printf("Error parsing StartTime: %v", err)
			continue
		}
		startTime := startTimeParsed.Unix()
		incoming := strconv.FormatFloat(flow.Incoming, 'f', -1, 64)
		outgoing := strconv.FormatFloat(flow.Outgoing, 'f', -1, 64)
		balance := strconv.FormatFloat(flow.Balance, 'f', -1, 64)
		// fmt.Printf("  %s: %+v\n", name, flow)
		StablecoinSMCMonth(startTime, incoming, outgoing, balance, name, flow.NameCoin)
	}
}
