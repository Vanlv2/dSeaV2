package main

import (
	"log"
<<<<<<< HEAD
	"main/services/stablecoin"
=======

	"main/services/bitcoinNetFlow"
>>>>>>> test
)

// BinanceView KrakenView, binance, kraken, models

func main() {
	log.Println("Starting to fetch Binance coin prices...")

<<<<<<< HEAD
	// go ohlcv.RunOHLCV()
	go stablecoin.Stablecoin()
	// go bitcoinNetFlow.BitcoinNetFlow()
=======
	// go narrativesPerforments.NarrativesPerforment()
	// go stablecoin.Stablecoin()
	go bitcoinNetFlow.BitcoinNetFlow()
>>>>>>> test

	select {}
	// WaitGroup để đảm bảo tất cả goroutines đều kết thúc an toàn
}
