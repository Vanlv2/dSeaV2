package main

import (
	"context"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
)

func GetChainData(chainName string) *ChainData {
	chainsLock.RLock()
	defer chainsLock.RUnlock()

	if data, exists := chains[chainName]; exists {
		return data
	}
	return nil
}

func InitChainData(chainName string) *ChainData {
	chainsLock.Lock()
	defer chainsLock.Unlock()

	if _, exists := chains[chainName]; !exists {
		chains[chainName] = &ChainData{
			LastProcessedBlock:  big.NewInt(0),
			ProcessedTxs:        make(map[string]bool),
			DisconnectedChannel: make(chan struct{}, 100),
			LogData:             make(map[string]interface{}),
			IsProcessingReorg:   false,
		}
	}

	return chains[chainName]
}

func handle_chain(chainName string) {
	chainData := InitChainData(chainName)

	load_config(chooseChain[chainName], chainName)

	client, err := ethclient.Dial(chainData.Config.WssRPC)
	if err != nil {
		log.Fatalf("Không thể kết nối đến RPC cho %s: %v", chainName, err)
	}
	defer client.Close()

	go func() {
		for range chainData.DisconnectedChannel {
			handle_disconnected_logs(client, chainName)
			log.Printf("Đã xử lý xong các log bị bỏ lỡ cho %s", chainName)
		}
	}()

	start_handle(client, chainName)

	select {}
}

func start_handle(client *ethclient.Client, chainName string) {
	ctx := context.Background()

	chainData := GetChainData(chainName)
	if chainData == nil {
		log.Fatalf("Không tìm thấy dữ liệu cho chain %s", chainName)
	}

	currentBlock, err := client.BlockNumber(context.Background())
	if err != nil {
		log.Fatalf("Không thể lấy số khối hiện tại cho chain %s: %v", chainName, err)
	}
	chainData.LastProcessedBlock = big.NewInt(int64(currentBlock))

	log.Printf("======= KHỞI ĐỘNG HỆ THỐNG CHO CHAIN %s =======", chainName)
	log.Printf("Khởi tạo lastProcessedBlock = %d", currentBlock)
	log.Printf("TransferSignature: %s", chainData.Config.TransferSignature)
	log.Printf("==================================")

	log.Printf("Bắt đầu xử lý logs theo thời gian thực cho chain %s...", chainName)
	go handle_logs(ctx, client, chainName)

	log.Printf("Bắt đầu giám sát logs bị bỏ lỡ cho chain %s...", chainName)
	go periodic_logs_monitoring(ctx, client, chainName)
}
