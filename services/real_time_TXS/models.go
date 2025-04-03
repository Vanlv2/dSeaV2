package real_time_TXS

import (
	"math/big"
	"sync"
)

type Config struct {
	RPC                   string `json:"rpc"`
	WssRPC                string `json:"wssRpc"`
	WssToken              string `json:"wssToken"`
	Chain                 string `json:"chain"`
	WrappedBTCAddress     string `json:"wrappedBTCAddress"`
	WrapWrappedBNBAddress string `json:"wrapWrappedBNBAddress"`
	TimeNeedToBlock       int    `json:"timeNeedToBlock"`
}

type ChainData struct {
	Config              Config
	LastProcessedBlock  *big.Int
	ProcessedTxs        map[string]bool
	DisconnectedChannel chan struct{}
	LogData             map[string]interface{}
	IsProcessingReorg   bool
}

var chains = make(map[string]*ChainData)
var chainsLock sync.RWMutex

var chooseChain = map[string]string{
	"bsc": "./config_chain/config-bsc.json",
}
var processLock sync.Mutex
