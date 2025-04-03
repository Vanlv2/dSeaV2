package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

// GlobalTransactions lưu trữ tất cả các giao dịch từ tất cả các chain
var GlobalTransactions = struct {
	sync.RWMutex
	Data     []map[string]interface{}
}{
	Data:     make([]map[string]interface{}, 0, 1000),
}

// AddTransaction thêm một giao dịch vào mảng toàn cục
func AddTransaction(txData map[string]interface{}) {
	GlobalTransactions.Lock()
	defer GlobalTransactions.Unlock()

	// Giới hạn kích thước mảng để tránh sử dụng quá nhiều bộ nhớ
	if len(GlobalTransactions.Data) >= 10000 {
		// Xóa 1000 giao dịch cũ nhất khi đạt giới hạn
		GlobalTransactions.Data = GlobalTransactions.Data[1000:]
	}

	GlobalTransactions.Data = append(GlobalTransactions.Data, txData)
}

// GetRecentTransactions trả về n giao dịch gần đây nhất
func GetRecentTransactions(n int) []map[string]interface{} {
	GlobalTransactions.RLock()
	defer GlobalTransactions.RUnlock()

	if len(GlobalTransactions.Data) == 0 {
		return []map[string]interface{}{}
	}

	if n > len(GlobalTransactions.Data) {
		n = len(GlobalTransactions.Data)
	}

	startIdx := len(GlobalTransactions.Data) - n
	return GlobalTransactions.Data[startIdx:]
}

func initialize_chain(chainName string) (*ethclient.Client, *ChainData) {
	configPath, exists := chooseChain[chainName]
	if !exists {
		return nil, nil
	}

	load_config(configPath, chainName)
	chainData := GetChainData(chainName)
	if chainData == nil {
		return nil, nil
	}

	client, err := ethclient.Dial(chainData.Config.RPC)
	if err != nil {
		return nil, nil
	}

	// Khởi tạo LastProcessedBlock bằng khối mới nhất
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	latestBlock, err := client.BlockNumber(ctx)
	if err == nil {
		chainData.LastProcessedBlock = big.NewInt(int64(latestBlock))
		fmt.Printf("✅ Khởi tạo LastProcessedBlock = %d cho chain %s\n",
			latestBlock, chainName)
	}

	return client, chainData
}

func handle_chain(ctx context.Context, chainName string) {
	// Khởi tạo kết nối và dữ liệu cho chain
	client, chainData := initialize_chain(chainName)
	if client == nil || chainData == nil {
		return
	}

	// Tạo context con cho mỗi chain
	chainCtx, chainCancel := context.WithCancel(ctx)
	defer chainCancel()

	// Khởi động các goroutine để xử lý logs
	go handle_logs(chainCtx, client, chainName)
	go periodic_logs_monitoring(chainCtx, client, chainName)

	// Xử lý các logs bị bỏ lỡ khi có tín hiệu
	for {
		select {
		case <-chainData.DisconnectedChannel:
			go handle_disconnected_logs(client, chainName)
		case <-chainCtx.Done():
			return
		}
	}
}

// In GlobalTransactions định kỳ
func printGlobalTransactions() {
	for {
		time.Sleep(5 * time.Second) // In cứ mỗi 5 giây

		GlobalTransactions.RLock()
		fmt.Println("=== GLOBAL TRANSACTIONS ===")
		fmt.Printf("Số lượng giao dịch: %d\n", len(GlobalTransactions.Data))

		// In 5 giao dịch gần nhất để xem
		numToPrint := 5
		if len(GlobalTransactions.Data) > 0 {
			startIdx := len(GlobalTransactions.Data)
			if startIdx > numToPrint {
				startIdx = startIdx - numToPrint
			} else {
				startIdx = 0
			}

			for i := startIdx; i < len(GlobalTransactions.Data); i++ {
				txJSON, _ := json.MarshalIndent(GlobalTransactions.Data[i], "", "  ")
				fmt.Printf("Giao dịch #%d:\n%s\n", i, string(txJSON))
			}
		} else {
			fmt.Println("Chưa có giao dịch nào được ghi nhận.")
		}
		fmt.Println("===========================")
		GlobalTransactions.RUnlock()
	}
}

func run_chains() {
	chains := []string{"bsc"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, chain := range chains {
		go handle_chain(ctx, chain)
	}

	// Giữ chương trình chạy
	select {}
}

func main() {
	InitTokenPrices()
	go printGlobalTransactions()
	run_chains()
	select {}
}
