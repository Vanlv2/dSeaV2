package fearGreedindex

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	feargreedindex "main/config/fearGreedindex"
)

func ConnectToSMC(FormData FormDataAdjusted) {
	// Kết nối WebSocket tới node BSC Testnet
	client, err := ethclient.Dial("wss://bsc-testnet-rpc.publicnode.com")
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	// Địa chỉ hợp đồng mà bạn muốn lắng nghe sự kiện
	contractAddr := common.HexToAddress(feargreedindex.ContractAddress)

	// Parse ABI
	contractABI, err := abi.JSON(strings.NewReader(feargreedindex.ContractABI))
	if err != nil {
		fmt.Printf("Error parsing ABI: %v\n", err)
	}

	// Private key của người gửi (dùng cho giao dịch)
	privateKey, err := crypto.HexToECDSA(feargreedindex.PrivateKey)
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}

	// Địa chỉ từ private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("Error casting public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Lấy nonce của tài khoản người gửi
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("Failed to get nonce: %v", err)
	}

	// Lấy gasPrice hiện tại
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("Failed to get gas price: %v", err)
	}

	// Dữ liệu được mã hóa cho hàm recordData
	data, err := contractABI.Pack("recordIndex", FormData)
	if err != nil {
		log.Fatalf("Failed to pack function call date: %v", err)
	}

	// Tạo giao dịch
	gasLimit, err := client.EstimateGas(context.Background(), ethereum.CallMsg{
		From: fromAddress,
		To:   &contractAddr,
		Data: data,
	})

	if err != nil {
		log.Fatalf("Failed to estimate gas: %v", err)
	}
	gasLimit = gasLimit * 12 / 10 // Add 20% buffer
	tx := types.NewTransaction(nonce, contractAddr, big.NewInt(0), gasLimit, gasPrice, data)

	// Ký giao dịch bằng private key
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatalf("Failed to get chain ID: %v", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}

	// Gửi giao dịch
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatalf("Failed to send transactionDate: %v", err)
	}
}
