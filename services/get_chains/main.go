package main

import (
	"main/services"
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
	go func() {
		// handle_cosmos_http()
		// handle_cosmos_ws()
		// handle_wss_vechain()
		// handle_http_vechain()
		// HandleTronHTTP()
		// handle_tron_ws()
		// services.Handle_stellar_http()
		// services.Handle_stellar_ws()
		// services.Handle_tezos_http()
		// services.Handle_tezos_ws()
		// services.Handle_algorand_http()
		// handle_http()
		// handle_ws()
		services.RunCryptoDataProcessor("./configs/config-binance.json")


	}()
	select {}
}
