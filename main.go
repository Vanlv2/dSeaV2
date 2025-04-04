package main

import (
	"log"

	// "main/services/bitcoinNetFlow"
	// "main/services/fearGreedindex"
	"main/services/entities"
	// getChains "main/services/get_chains"
	// "main/services/ohlcv"
	// "main/services/stablecoin"
	// Onchain_exchange_flow "main/services/Onchain_exchange_flow"
	// getChains "main/services/get_chains"
	// real_time_txs "main/services/real_time_TXS"
)

func main() {
	log.Println("Starting to fetch Binance coin prices...")

	//done
	// go ohlcv.RunOHLCV()
	// go stablecoin.Stablecoin()
	// go fearGreedindex.FearGreedindex()
	// go bitcoinNetFlow.BitcoinNetFlow()
	// getChains.StartGetChains()
	entities.Entities()
	// Onchain_exchange_flow.Onchain_exchange_flow()
	// real_time_txs.Real_time_txs()

	select {}
}
