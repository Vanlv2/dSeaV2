package bitcoinNetFlow

import (
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
