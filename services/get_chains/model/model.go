package model

import (
	"sync"
	"time"
)

// ChainDataVan interface chung cho tất cả chuỗi
type ChainDataVan interface {
	GetLastProcessedBlockVan() int64
	SetLastProcessedBlockVan(int64)
	GetConfigVan() interface{}
	SetConfigVan(interface{})
	GetLogDataVan() map[string]interface{}
}

// ConfigTezos cấu hình cho Tezos
type ConfigTezos struct {
	RPC   string `json:"rpc"`
	Chain string `json:"chain"`
}

// ConfigElrond cấu hình cho Elrond
type ConfigElrond struct {
	API string `json:"api"`
}

// ConfigAlgorand cấu hình cho Algorand
type ConfigAlgorand struct {
	API string `json:"api"`
}

// ConfigStellar cấu hình cho Stellar
type ConfigStellar struct {
	HorizonURL string `json:"horizon_url"`
	Network    string `json:"network"` // "public" hoặc "testnet"
}

// ChainDataTezos cho Tezos
type ChainDataTezos struct {
	LastProcessedBlock int64
	Config             ConfigTezos
	LogData            map[string]interface{}
}

func (c *ChainDataTezos) GetLastProcessedBlockVan() int64       { return c.LastProcessedBlock }
func (c *ChainDataTezos) SetLastProcessedBlockVan(n int64)      { c.LastProcessedBlock = n }
func (c *ChainDataTezos) GetConfigVan() interface{}             { return c.Config }
func (c *ChainDataTezos) SetConfigVan(cfg interface{})          { c.Config = cfg.(ConfigTezos) }
func (c *ChainDataTezos) GetLogDataVan() map[string]interface{} { return c.LogData }

// ChainDataElrond cho Elrond
type ChainDataElrond struct {
	LastProcessedBlock int64
	Config             ConfigElrond
	LogData            map[string]interface{}
}

func (c *ChainDataElrond) GetLastProcessedBlockVan() int64       { return c.LastProcessedBlock }
func (c *ChainDataElrond) SetLastProcessedBlockVan(n int64)      { c.LastProcessedBlock = n }
func (c *ChainDataElrond) GetConfigVan() interface{}             { return c.Config }
func (c *ChainDataElrond) SetConfigVan(cfg interface{})          { c.Config = cfg.(ConfigElrond) }
func (c *ChainDataElrond) GetLogDataVan() map[string]interface{} { return c.LogData }

// ChainDataAlgorand cho Algorand
type ChainDataAlgorand struct {
	LastProcessedBlock int64
	Config             ConfigAlgorand
	LogData            map[string]interface{}
}

func (c *ChainDataAlgorand) GetLastProcessedBlockVan() int64       { return c.LastProcessedBlock }
func (c *ChainDataAlgorand) SetLastProcessedBlockVan(n int64)      { c.LastProcessedBlock = n }
func (c *ChainDataAlgorand) GetConfigVan() interface{}             { return c.Config }
func (c *ChainDataAlgorand) SetConfigVan(cfg interface{})          { c.Config = cfg.(ConfigAlgorand) }
func (c *ChainDataAlgorand) GetLogDataVan() map[string]interface{} { return c.LogData }

// ChainDataStellar cho Stellar
type ChainDataStellarVan struct {
	LastProcessedLedger int64 // Sử dụng Ledger thay vì Block cho Stellar
	Config              ConfigStellar
	LogData             map[string]interface{}
}

func (c *ChainDataStellarVan) GetLastProcessedBlockVan() int64       { return c.LastProcessedLedger }
func (c *ChainDataStellarVan) SetLastProcessedBlockVan(n int64)      { c.LastProcessedLedger = n }
func (c *ChainDataStellarVan) GetConfigVan() interface{}             { return c.Config }
func (c *ChainDataStellarVan) SetConfigVan(cfg interface{})          { c.Config = cfg.(ConfigStellar) }
func (c *ChainDataStellarVan) GetLogDataVan() map[string]interface{} { return c.LogData }

// TransactionRecordVan định nghĩa bản ghi giao dịch
type TransactionRecordVan struct {
	BlockHeight string    `json:"block_height"`
	BlockHash   string    `json:"block_hash"`
	BlockTime   time.Time `json:"block_time"`
	ChainID     string    `json:"chain_id"`
	TxHash      string    `json:"tx_hash"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Amount      string    `json:"amount"`
	Token       string    `json:"token"`
	TotalAmount string    `json:"total_amount"`
	TxType      string    `json:"tx_type"`
	Timestamp   string    `json:"timestamp"`
}

// BlockResponseVan cho Tezos RPC
type BlockResponseVan struct {
	Protocol string `json:"protocol"`
	ChainID  string `json:"chain_id"`
	Hash     string `json:"hash"`
	Header   struct {
		Level     int64     `json:"level"`
		Timestamp time.Time `json:"timestamp"`
	} `json:"header"`
	Operations [][]struct {
		Contents []struct {
			Kind        string `json:"kind"`
			Source      string `json:"source,omitempty"`
			Destination string `json:"destination,omitempty"`
			Amount      string `json:"amount,omitempty"`
		} `json:"contents"`
	} `json:"operations"`
}

// ElrondBlockResponseVan định nghĩa cấu trúc block từ MultiversX API
type ElrondBlockResponseVan struct {
	Hash              string `json:"hash"`
	Epoch             int64  `json:"epoch"`
	Nonce             int64  `json:"nonce"`
	PrevHash          string `json:"prevHash"`
	Proposer          string `json:"proposer"`
	PubKeyBitmap      string `json:"pubKeyBitmap"`
	Round             int64  `json:"round"`
	Shard             uint32 `json:"shard"`
	Size              int64  `json:"size"`
	SizeTxs           int64  `json:"sizeTxs"`
	StateRootHash     string `json:"stateRootHash"`
	ScheduledRootHash string `json:"scheduledRootHash,omitempty"`
	Timestamp         int64  `json:"timestamp"`
	TxCount           int64  `json:"txCount"`
	GasConsumed       int64  `json:"gasConsumed"`
	GasRefunded       int64  `json:"gasRefunded"`
	GasPenalized      int64  `json:"gasPenalized"`
	MaxGasLimit       int64  `json:"maxGasLimit"`
}

// ElrondTransactionVan định nghĩa cấu trúc giao dịch từ MultiversX API
type ElrondTransactionVan struct {
	Hash     string `json:"hash"`
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Value    int64  `json:"value"`
}

// AlgorandBlockResponseVan định nghĩa cấu trúc block từ Algorand API
type AlgorandBlockResponseVan struct {
	Hash              string                   `json:"hash"`
	PreviousBlockHash string                   `json:"previous-block-hash"`
	Round             int64                    `json:"round"`
	Timestamp         int64                    `json:"timestamp"`
	Rewards           int64                    `json:"rewards"`
	FeeSink           string                   `json:"fee-sink"`
	RewardPool        string                   `json:"reward-pool"`
	GenesisID         string                   `json:"genesis-id"`   // Thêm GenesisID
	GenesisHash       string                   `json:"genesis-hash"` // Thêm GenesisHash
	Transactions      []AlgorandTransactionVan `json:"transactions"` // Thêm danh sách giao dịch
}

// Biến toàn cục
var ChainDataMapVan = make(map[string]ChainDataVan) // Sử dụng interface
var LogMutexVan sync.Mutex
var ProcessLockVan sync.Mutex

// AlgorandTransactionVan đại diện cho một giao dịch Algorand
type AlgorandTransactionVan struct {
	ID                       string                       `json:"id"`
	Sender                   string                       `json:"sender"`
	Fee                      int64                        `json:"fee"`
	FirstValid               int64                        `json:"first-valid"`
	LastValid                int64                        `json:"last-valid"`
	Type                     string                       `json:"tx-type"`
	Signature                map[string]interface{}       `json:"signature,omitempty"`
	ConfirmedRound           int64                        `json:"confirmed-round,omitempty"`
	IntraRoundOffset         int64                        `json:"intra-round-offset,omitempty"`
	RoundTime                int64                        `json:"round-time,omitempty"`
	CloseRewards             int64                        `json:"close-rewards,omitempty"`
	ClosingAmount            int64                        `json:"closing-amount,omitempty"`
	ReceiverRewards          int64                        `json:"receiver-rewards,omitempty"`
	SenderRewards            int64                        `json:"sender-rewards,omitempty"`
	GenesisHash              string                       `json:"genesis-hash,omitempty"`
	GenesisID                string                       `json:"genesis-id,omitempty"`
	PaymentTransaction       *PaymentTransactionVan       `json:"payment-transaction,omitempty"`        // Sử dụng struct cụ thể thay vì map
	AssetTransferTransaction *AssetTransferTransactionVan `json:"asset-transfer-transaction,omitempty"` // Thêm AssetTransferTransaction
}

// PaymentTransactionVan định nghĩa chi tiết cho payment transaction
type PaymentTransactionVan struct {
	Amount      int64  `json:"amount"`
	Receiver    string `json:"receiver"`
	CloseAmount int64  `json:"close-amount,omitempty"`
	CloseTo     string `json:"close-to,omitempty"`
}

// AssetTransferTransactionVan định nghĩa chi tiết cho asset transfer transaction
type AssetTransferTransactionVan struct {
	AssetID     int64  `json:"asset-id"`
	Amount      int64  `json:"amount"`
	Receiver    string `json:"receiver"`
	Sender      string `json:"sender,omitempty"`
	CloseTo     string `json:"close-to,omitempty"`
	CloseAmount int64  `json:"close-amount,omitempty"`
}

// StellarLedgerResponse định nghĩa cấu trúc ledger từ Stellar Horizon API
type StellarLedgerResponseVan struct {
	ID               string    `json:"id"`
	Sequence         int64     `json:"sequence"` // Tương đương block height
	Hash             string    `json:"hash"`
	PrevHash         string    `json:"prev_hash"`
	Timestamp        time.Time `json:"closed_at"` // Thời gian đóng ledger
	TransactionCount int64     `json:"transaction_count"`
}

// StellarTransaction định nghĩa giao dịch từ Stellar Horizon API
type StellarTransactionVan struct {
	ID          string    `json:"id"`
	Hash        string    `json:"hash"`
	Ledger      int64     `json:"ledger"`
	Source      string    `json:"source_account"`
	Destination string    `json:"destination,omitempty"`
	Amount      string    `json:"amount,omitempty"`
	AssetType   string    `json:"asset_type,omitempty"`
	AssetCode   string    `json:"asset_code,omitempty"`
	AssetIssuer string    `json:"asset_issuer,omitempty"`
	Timestamp   time.Time `json:"created_at"`
}
type TransactionEventVan struct {
	Address         string `json:"address"`
	Amount          string `json:"amount"`
	BlockNumber     int64  `json:"block_number"`
	EventSignature  string `json:"event_signature"`
	FromAddress     string `json:"from_address"`
	LogIndex        int    `json:"log_index"`
	NameChain       string `json:"name_chain"`
	RawData         string `json:"raw_data"`
	Timestamp       string `json:"timestamp"`
	ToAddress       string `json:"to_address"`
	TransactionType string `json:"transaction_type"`
	TxHash          string `json:"tx_hash"`
}
