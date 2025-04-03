package main

import (
	"log"

	"main/services/bitcoinNetFlow"
)

func main() {
	log.Println("Starting to fetch Binance coin prices...")

	// go narrativesPerforments.NarrativesPerforment()
	// go stablecoin.Stablecoin()
	go bitcoinNetFlow.BitcoinNetFlow()

	select {}
	// WaitGroup để đảm bảo tất cả goroutines đều kết thúc an toàn
}
