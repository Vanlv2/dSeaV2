package smartcontract

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	structData "main/services/bitcoinNetFlow/smart_contract"
	Time "main/services/bitcoinNetFlow/smart_contract/methods"
)

// CallContractMethod gọi phương thức của contract với các tham số tùy chỉnh
func CallContractMethodDaily(config structData.ConfigContract, methodName string, params map[string]interface{}) error {
	// Kết nối với BSC node
	client, err := ethclient.Dial(config.BscNodeURL)
	if err != nil {
		return fmt.Errorf("failed to connect to BSC node: %v", err)
	}

	// Tải private key
	privateKey, err := crypto.HexToECDSA(config.PrivateKeyHex)
	if err != nil {
		return fmt.Errorf("invalid private key: %v", err)
	}

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	address := crypto.PubkeyToAddress(*publicKey)

	fmt.Printf("Địa chỉ ví của bạn: %s\n", address.Hex())

	// Phân tích ABI
	parsedABI, err := abi.JSON(strings.NewReader(config.ContractABI))
	if err != nil {
		return fmt.Errorf("failed to parse contract ABI: %v", err)
	}

	// Lấy nonce hiện tại cho địa chỉ
	nonce, err := client.PendingNonceAt(context.Background(), address)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %v", err)
	}

	// Lấy giá gas hiện tại
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to suggest gas price: %v", err)
	}

	// Chuẩn bị tham số cho phương thức
	methodArgs, err := Time.PrepareMethodArgsDaily(methodName, params)
	if err != nil {
		return err
	}

	// Đóng gói dữ liệu cho phương thức
	data, err := parsedABI.Pack(methodName, methodArgs...)
	if err != nil {
		return fmt.Errorf("failed to pack data for method %s: %v", methodName, err)
	}

	// Tạo transaction
	gasLimit := uint64(300000) // Đặt giới hạn gas phù hợp
	tx := types.NewTransaction(
		nonce,
		common.HexToAddress(config.ContractAddress),
		big.NewInt(0), // Không gửi ETH/BNB cùng với transaction
		gasLimit,
		gasPrice,
		data,
	)

	// Lấy chainID
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %v", err)
	}

	// Ký transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Gửi transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %v", err)
	}

	fmt.Printf("Transaction đã được gửi: %s\n", signedTx.Hash().Hex())
	fmt.Printf("Đã gọi hàm %s\n", methodName)

	// Đợi transaction được xác nhận
	fmt.Println("Đang đợi transaction được xác nhận...")
	receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		return fmt.Errorf("failed to wait for transaction confirmation: %v", err)
	}

	for param, value := range params {
		fmt.Printf("Tham số: %s, Giá trị: %v\n", param, value)
	}

	if receipt.Status == 1 {
		fmt.Println("Transaction đã được xác nhận thành công!")
	} else {
		fmt.Println("Transaction thất bại!")
	}

	return nil
}
