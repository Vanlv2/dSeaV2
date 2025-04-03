package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"math/big"
	"time"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/mr-tron/base58"

	"main/configs"
)

type Transaction struct {
	Hash        string
	From        string
	To          string
	Value       *big.Int
	Token       string
	BlockNumber uint64
}

func saveSolanaTransaction(db *sql.DB, tx Transaction) error {
	query := `
		INSERT INTO solana_transactions (hash, from_address, to_address, value, token, block_number)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query, tx.Hash, tx.From, tx.To, tx.Value.String(), tx.Token, tx.BlockNumber)
	return err
}

func HandleChainSolana(db *sql.DB, fileConfig string, stopChan <-chan struct{}, logger *log.Logger, txChan chan<- interface{}) {
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
			block, err := c.GetBlock(context.Background(), latestSlot)
			if err != nil {
				logger.Printf("Failed to get block: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
			logBlockTransactions(db, block, latestSlot, txChan, logger)
			time.Sleep(time.Duration(config.TimeNeedToBlock) * time.Millisecond)
		}
	}
}

func logBlockTransactions(db *sql.DB, block *client.Block, slot uint64, txChan chan<- interface{}, logger *log.Logger) {
	logger.Printf("Block at slot #%d has %d transactions", slot, len(block.Transactions))

	for _, tx := range block.Transactions {
		if len(tx.Transaction.Message.Accounts) < 2 {
			logger.Printf("Skipping transaction with insufficient accounts: %d", len(tx.Transaction.Message.Accounts))
			continue
		}

		sender := tx.Transaction.Message.Accounts[0].ToBase58()
		receiver := tx.Transaction.Message.Accounts[1].ToBase58()
		preBalanceSender := tx.Meta.PreBalances[0]
		postBalanceSender := tx.Meta.PostBalances[0]
		transferredAmount := preBalanceSender - postBalanceSender

		hash := ""
		if len(tx.Transaction.Signatures) > 0 {
			hash = base58.Encode(tx.Transaction.Signatures[0][:])
		}
		txData := Transaction{
			Hash:        hash,
			From:        sender,
			To:          receiver,
			Value:       big.NewInt(transferredAmount),
			Token:       "SOL",
			BlockNumber: slot,
		}

		if err := saveSolanaTransaction(db, txData); err != nil {
			logger.Printf("Failed to save transaction: %v", err)
		} else {
			logger.Printf("Transaction saved: %s", hash)
		}

		txJSON, _ := json.MarshalIndent(txData, "", "  ")
		logger.Printf("Transaction JSON: %s", string(txJSON))
	}
}
