package main

import (
	"log"

	getChains "main/services/get_chains"
)

func main() {
	log.Println("Starting to fetch Binance coin prices...")

	// go narrativesPerforments.NarrativesPerforment()
	// go stablecoin.Stablecoin()
	// go bitcoinNetFlow.BitcoinNetFlow()
	getChains.StartGetChains()

	select {}
	// WaitGroup để đảm bảo tất cả goroutines đều kết thúc an toàn
}
