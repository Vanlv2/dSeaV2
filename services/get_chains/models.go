package get_chains

import (
	"math/big"
	"sync"
	"time"
)

type Config struct {
	RPC                 string `json:"rpc"`
	WssRPC              string `json:"wssRpc"`
	WssToken            string `json:"wssToken"`
	TransferSignature   string `json:"transferSignature"`
	Chain               string `json:"chain"`
	EthContractAddress  string `json:"ethContractAddress"`
	UsdtContractAddress string `json:"usdtContractAddress"`
	UsdcContractAddress string `json:"usdcContractAddress"`
	WrappedBTCAddress   string `json:"wrappedBTCAddress"`
	TimeNeedToBlock     int    `json:"timeNeedToBlock"`
}


type ChainData struct {
	Config              Config
	LastProcessedBlock  *big.Int
	ProcessedTxs        map[string]bool
	DisconnectedChannel chan struct{}
	LogData             map[string]interface{}
	IsProcessingReorg   bool
}

type Transaction struct {
	ID              int64
	Timestamp       time.Time
	BlockNumber     uint64
	TxHash          string
	Address         string
	Amount          string
	RawData         string
	EventSignature  string
	FromAddress     string
	ToAddress       string
	NameChain       string
	TransactionType string
}

type WebSocketMessage struct {
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params"`
}

type WebSocketResponse struct {
	Action  string      `json:"action"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
}

var chains = make(map[string]*ChainData)
var chainsLock sync.RWMutex

var chooseChain = map[string]string{
	"ethereum":  "./services/get_chains/config_chain/config-ethereum.json",
	"bsc":       "./services/get_chains/config_chain/config-bsc.json",
	"avalanche": "./services/get_chains/config_chain/config-avalanche.json",
	"polygon":   "./services/get_chains/config_chain/config-polygon.json",
	"arbitrum":  "./services/get_chains/config_chain/config-arbitrum.json",
	"optimism":  "./services/get_chains/config_chain/config-optimism.json",
	"fantom":    "./services/get_chains/config_chain/config-fantom.json",
	"base":      "./services/get_chains/config_chain/config-base.json",
	"tron":      "./services/get_chains/config_chain/config-tron.json",
}
var processLock sync.Mutex
