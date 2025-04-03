package services

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/blocto/solana-go-sdk/client"

	"main/services/get_chains/configs"
)

type Transaction struct {
	Hash        string
	From        string
	To          string
	Value       *big.Int
	Token       string
	BlockNumber uint64
}

func HandleChainSolana(fileConfig string, stopChan <-chan struct{}, logger *log.Logger, txChan chan<- interface{}) {
	config, err := configs.LoadConfigLang(fileConfig)
	if err != nil {
		logger.Printf("Failed to load config: %v", err)
		return
	}

	c := client.NewClient(config.RPC)

	for {
		select {
		case <-stopChan:
			logger.Println("Stopping Solana chain")
			return
		default:
			latestSlot, err := c.GetSlot(context.Background())
			if err != nil {
				logger.Printf("Failed to get latest slot: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			logger.Printf("Latest slot: %d", latestSlot)
			time.Sleep(time.Duration(config.TimeNeedToBlock) * time.Millisecond)
		}
	}
}