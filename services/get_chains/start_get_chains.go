package get_chains

import (
	"log"
	"os"
	"sync"

	"main/services/get_chains/services"
)

func handle_ws() {
	chains := []string{
		"ethereum",
		"bsc",
		"avalanche",
		"polygon",
		"arbitrum",
		"optimism",
		"fantom",
		"base",
	}

	for _, chain := range chains {
		go handle_chain(chain)
	}
}

// Hàm mới để xử lý Solana HTTP
func handle_solana_http() {
	logger := log.New(os.Stdout, "[SOLANA-HTTP] ", log.LstdFlags)
	stopChan := make(chan struct{})
	txChan := make(chan interface{})
	
	// Xử lý các giao dịch nhận được (có thể thêm logic xử lý ở đây)
	go func() {
		for tx := range txChan {
			logger.Printf("Received transaction: %v", tx)
		}
	}()
	
	// Gọi hàm xử lý Solana HTTP
	services.HandleChainSolana("./services/get_chains/configs/config-sol.json", stopChan, logger, txChan)
}

// Hàm mới để xử lý Bitcoin và Solana qua WebSocket
func handle_btc_sol_ws() {
	// Gọi hàm xử lý dữ liệu tiền điện tử
	services.HandleRealTimeCrypto("./services/get_chains/configs/config-binance.json", make(chan struct{}))
}

func StartGetChains() {
	var wg sync.WaitGroup
	
	// Danh sách tất cả các dịch vụ cần chạy
	services := []func(){
		handle_cosmos_http,
		handle_cosmos_ws,
		handle_wss_vechain,
		handle_http_vechain,
		HandleTronHTTP,
		handle_tron_ws,
		services.Handle_stellar_http,
		services.Handle_stellar_ws,
		services.Handle_tezos_http,
		services.Handle_tezos_ws,
		services.Handle_algorand_http,
		handle_http,
		handle_ws,
		func() { services.RunCryptoDataProcessor("./configs/config-binance.json") },
		// Thêm các hàm mới vào danh sách dịch vụ
		handle_solana_http,
		handle_btc_sol_ws,
	}

	for _, service := range services {
		wg.Add(1)
		go func(serviceFunc func()) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in service: %v", r)
				}
			}()
			serviceFunc()
		}(service)
	}

	wg.Wait()
}
