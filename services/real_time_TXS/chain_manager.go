package real_time_TXS

import (
	"math/big"
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
