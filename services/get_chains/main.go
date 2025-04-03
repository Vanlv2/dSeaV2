package main

import (
	"log"
	"main/services"
	"sync"
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

func main() {
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
