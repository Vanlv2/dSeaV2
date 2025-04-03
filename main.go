package main

import (
	"log"
	"main/services/get_chains"
)

func main() {
	log.Println("Starting to fetch Binance coin prices...")

	//done
	// go ohlcv.RunOHLCV()
	// go stablecoin.Stablecoin()
	// go fearGreedindex.FearGreedindex()
	// go bitcoinNetFlow.BitcoinNetFlow()
	go get_chains.StartGetChains()

	select {}
}
