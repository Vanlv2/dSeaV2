package main

import (
	"log"
	"main/services/stablecoin"
)

// BinanceView KrakenView, binance, kraken, models

func main() {
	log.Println("Starting to fetch Binance coin prices...")

	// go ohlcv.RunOHLCV()
	go stablecoin.Stablecoin()
	// go bitcoinNetFlow.BitcoinNetFlow()

	select {}
	// WaitGroup để đảm bảo tất cả goroutines đều kết thúc an toàn
}
