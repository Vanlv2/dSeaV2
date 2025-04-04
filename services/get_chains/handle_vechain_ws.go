package get_chains

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// Cấu trúc dữ liệu cho khối
type BlockVechainWS struct {
	ID           string   `json:"id"`
	Number       uint32   `json:"number"`
	ParentID     string   `json:"parentID"`
	Timestamp    uint64   `json:"timestamp"`
	Transactions []string `json:"transactions"`
	Size         uint32   `json:"size"`
	GasLimit     uint64   `json:"gasLimit"`
	GasUsed      uint64   `json:"gasUsed"`
	TotalScore   uint64   `json:"totalScore"`
	TxsRoot      string   `json:"txsRoot"`
	StateRoot    string   `json:"stateRoot"`
	Beneficiary  string   `json:"beneficiary"`
}

// Cấu trúc dữ liệu cho giao dịch
type TransactionVechainWS struct {
	ID           string         `json:"id"`
	Origin       string         `json:"origin"`
	Gas          uint64         `json:"gas"`
	Clauses      []ClauseWS     `json:"clauses"`
	Size         uint32         `json:"size"`
	GasPriceCoef uint8          `json:"gasPriceCoef"`
	ChainTag     uint8          `json:"chainTag,omitempty"`
	BlockRef     string         `json:"blockRef,omitempty"`
	Expiration   uint32         `json:"expiration,omitempty"`
	DependsOn    string         `json:"dependsOn,omitempty"`
	Nonce        string         `json:"nonce,omitempty"`
	Meta         *MetaVechainWS `json:"meta,omitempty"`
}

// Cấu trúc dữ liệu cho meta của giao dịch
type MetaVechainWS struct {
	BlockID        string `json:"blockID"`
	BlockNumber    uint32 `json:"blockNumber"`
	BlockTimestamp uint64 `json:"blockTimestamp"`
}

// Cấu trúc dữ liệu cho mệnh đề trong giao dịch
type ClauseWS struct {
	To    string `json:"to"`
	Value string `json:"value"`
	Data  string `json:"data"`
}

// Cấu trúc cho log giao dịch
type TransactionLogVechainWS struct {
	BlockHeight     string `json:"block_height"`
	BlockHash       string `json:"block_hash"`
	BlockTime       string `json:"block_time"`
	ChainID         string `json:"chain_id"`
	TxHash          string `json:"tx_hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	ValueDecimal    string `json:"value_decimal,omitempty"`
	Gas             string `json:"gas"`
	GasPriceCoef    string `json:"gas_price_coef"`
	Size            string `json:"size"`
	TxType          string `json:"tx_type"`
	Method          string `json:"method,omitempty"`
	MethodSignature string `json:"method_signature,omitempty"`
	Nonce           string `json:"nonce,omitempty"`
}

// Logger để ghi log ra file
var fileLogger *log.Logger

// Thiết lập logger
func setupLoggerVechainWS() {
	// Tạo file log với tên chứa timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFile, err := os.Create(fmt.Sprintf("./services/get_chains/vechain_blocks_%s.log", timestamp))
	if err != nil {
		log.Fatal("Không thể tạo file log: ", err)
	}

	// Khởi tạo logger ghi vào file
	fileLogger = log.New(logFile, "", log.Ldate|log.Ltime)
	fileLogger.Println("=== Bắt đầu ghi log VeChain Blocks và Transactions ===")
}

// Lấy dữ liệu giao dịch từ API REST
func getTransactionVechainWS(txID string, nodeURL string) (*TransactionVechainWS, error) {
	apiURL := fmt.Sprintf("%s/transactions/%s", nodeURL, txID)

	log.Printf("Đang gọi API: %s", apiURL)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	// Kiểm tra nếu response không phải null
	if string(bodyBytes) != "null" && resp.StatusCode == http.StatusOK {
		var tx TransactionVechainWS
		if err := json.Unmarshal(bodyBytes, &tx); err != nil {
			return nil, err
		}
		return &tx, nil
	}

	return nil, fmt.Errorf("không thể lấy thông tin giao dịch: status %d", resp.StatusCode)
}

// Chuyển đổi giá trị hex sang decimal có thể đọc được
func hexToDecimalValueVechainWS(hexValue string) string {
	// Bỏ prefix "0x"
	hexValue = strings.TrimPrefix(hexValue, "0x")

	// Nếu chuỗi trống, trả về "0"
	if hexValue == "" {
		return "0"
	}

	// Chuyển đổi từ hex sang decimal
	bigInt := new(big.Int)
	bigInt, success := bigInt.SetString(hexValue, 16)
	if !success {
		return "Không thể chuyển đổi"
	}

	// Định dạng số với dấu phân cách hàng nghìn
	return formatBigIntVechainWS(bigInt)
}

// Định dạng số lớn với dấu phân cách hàng nghìn
func formatBigIntVechainWS(bigInt *big.Int) string {
	// Chuyển thành chuỗi
	numStr := bigInt.String()

	// Thêm dấu phân cách hàng nghìn
	var result strings.Builder
	len := len(numStr)

	for i, char := range numStr {
		result.WriteRune(char)
		if (len-i-1)%3 == 0 && i < len-1 {
			result.WriteRune(',')
		}
	}

	return result.String()
}

// Phân tích dữ liệu (data) của clause để xác định loại giao dịch
func determineTransactionTypeVechainWS(data string) string {
	if data == "0x" {
		return "simple_transfer"
	}

	if len(data) >= 10 {
		methodID := data[:10]

		// Một số loại phổ biến
		switch methodID {
		case "0xa9059cbb":
			return "token_transfer"
		case "0x23b872dd":
			return "token_transfer_from"
		case "0x095ea7b3":
			return "token_approve"
		case "0x42842e0e", "0xb88d4fde":
			return "nft_transfer"
		case "0x1249c58b":
			return "mint"
		case "0x18160ddd":
			return "totalSupply"
		case "0x70a08231":
			return "balanceOf"
		case "0x8d69446d":
			return "contract_interaction"
		}
	}

	return "contract_call"
}

// Lấy mô tả phương thức từ methodID
func getMethodNameVechainWS(methodID string) (string, string) {
	// Mapping cho các phương thức phổ biến trong các hợp đồng
	methodSignatures := map[string]string{
		// ERC20
		"0xa9059cbb": "transfer(address,uint256)",
		"0x23b872dd": "transferFrom(address,address,uint256)",
		"0x095ea7b3": "approve(address,uint256)",
		"0xd0e30db0": "deposit()",
		"0x2e1a7d4d": "withdraw(uint256)",
		"0x70a08231": "balanceOf(address)",
		"0x18160ddd": "totalSupply()",
		"0x313ce567": "decimals()",
		"0x06fdde03": "name()",
		"0x95d89b41": "symbol()",
		"0xdd62ed3e": "allowance(address,address)",
		"0x40c10f19": "mint(address,uint256)",
		"0x42966c68": "burn(uint256)",
		"0x79cc6790": "burnFrom(address,uint256)",
		// ERC721
		"0x42842e0e": "safeTransferFrom(address,address,uint256)",
		"0xb88d4fde": "safeTransferFrom(address,address,uint256,bytes)",
		"0x6352211e": "ownerOf(uint256)",
		"0x081812fc": "getApproved(uint256)",
		"0xe985e9c5": "isApprovedForAll(address,address)",
		"0xa22cb465": "setApprovalForAll(address,bool)",
		"0x1249c58b": "mint()",
		// VeChain specific
		"0x8d69446d": "callContract(address,bytes)",
		"0xbc1c58d1": "delegateAssets(address,uint256,bytes32)",
		"0x43a0d066": "transferToken(address,address,uint256)",
	}

	// Tên ngắn gọn cho phương thức
	methodNames := map[string]string{
		// ERC20
		"0xa9059cbb": "transfer",
		"0x23b872dd": "transferFrom",
		"0x095ea7b3": "approve",
		"0xd0e30db0": "deposit",
		"0x2e1a7d4d": "withdraw",
		"0x70a08231": "balanceOf",
		"0x18160ddd": "totalSupply",
		"0x313ce567": "decimals",
		"0x06fdde03": "name",
		"0x95d89b41": "symbol",
		"0xdd62ed3e": "allowance",
		"0x40c10f19": "mint",
		"0x42966c68": "burn",
		"0x79cc6790": "burnFrom",
		// ERC721
		"0x42842e0e": "safeTransferFrom",
		"0xb88d4fde": "safeTransferFrom",
		"0x6352211e": "ownerOf",
		"0x081812fc": "getApproved",
		"0xe985e9c5": "isApprovedForAll",
		"0xa22cb465": "setApprovalForAll",
		"0x1249c58b": "mint",
		// VeChain specific
		"0x8d69446d": "callContract",
		"0xbc1c58d1": "delegateAssets",
		"0x43a0d066": "transferToken",
	}

	signature := methodSignatures[methodID]
	if signature == "" {
		signature = methodID
	}

	name := methodNames[methodID]
	if name == "" {
		name = methodID
	}

	return name, signature
}


func handle_wss_vechain() {
	// Thiết lập logger
	setupLoggerVechainWS()

	// Thông tin kết nối - VeChain mainnet
	veChainNodeHost := "mainnet.veblocks.net"
	veChainNodeWSPort := "443"
	veChainNodeRESTURL := "https://" + veChainNodeHost

	// URL WebSocket cho đăng ký khối mới
	u := url.URL{
		Scheme: "wss",
		Host:   veChainNodeHost + ":" + veChainNodeWSPort,
		Path:   "/subscriptions/block",
	}

	log.Printf("Đang kết nối đến %s", u.String())
	fileLogger.Printf("Đang kết nối đến %s", u.String())

	// Thiết lập header
	header := http.Header{}
	header.Add("Origin", "https://"+veChainNodeHost)

	// Kết nối WebSocket
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 45 * time.Second

	c, _, err := dialer.Dial(u.String(), header)
	if err != nil {
		alternativeEndpoints := []string{
			"wss://mainnet.veblocks.net/subscriptions/beat",
			"wss://sync-mainnet.vechain.org/subscriptions/block",
		}

		var connected bool
		for _, endpoint := range alternativeEndpoints {
			log.Printf("Thử kết nối đến endpoint thay thế: %s", endpoint)
			alt, _, altErr := dialer.Dial(endpoint, header)
			if altErr == nil {
				log.Printf("Kết nối thành công đến %s", endpoint)
				c = alt
				connected = true
				break
			}
		}

		if !connected {
			log.Fatal("Không thể kết nối đến bất kỳ endpoint WebSocket nào!")
		}
	}
	defer c.Close()

	// Thông báo kết nối thành công
	log.Printf("Kết nối WebSocket thành công, đang chờ khối mới...")
	fileLogger.Printf("Kết nối WebSocket thành công, đang chờ khối mới...")

	// Đăng ký để nhận thông báo khi có gián đoạn
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Kênh để nhận tin nhắn
	done := make(chan struct{})

	// Goroutine để đọc tin nhắn
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("Lỗi đọc tin nhắn: ", err)
				return
			}

			log.Printf("Nhận được dữ liệu khối mới")

			// Xử lý thông tin khối nhận được
			var block BlockVechainWS
			if err := json.Unmarshal(message, &block); err != nil {
				log.Printf("Lỗi phân tích JSON: %v", err)
				continue
			}

			txCount := len(block.Transactions)
			blockTime := time.Unix(int64(block.Timestamp), 0).UTC()
			formattedBlockTime := blockTime.Format(time.RFC3339Nano)

			log.Printf("Nhận được khối #%d, ID: %s, có %d giao dịch",
				block.Number, block.ID, txCount)

			// In thông tin khối
			fileLogger.Printf("=== THÔNG TIN KHỐI #%d ===", block.Number)
			blockInfo := map[string]interface{}{
				"block_height": fmt.Sprintf("%d", block.Number),
				"block_hash":   block.ID,
				"block_time":   formattedBlockTime,
				"chain_id":     "vechain",
				"parent_id":    block.ParentID,
				"gas_limit":    block.GasLimit,
				"gas_used":     block.GasUsed,
				"total_score":  block.TotalScore,
				"beneficiary":  block.Beneficiary,
				"size":         block.Size,
				"tx_count":     txCount,
			}

			// Định dạng JSON đẹp
			blockJSON, _ := json.MarshalIndent(blockInfo, "", "    ")
			fileLogger.Println(string(blockJSON))

			// Xử lý các giao dịch trong khối
			if txCount > 0 {
				for _, txID := range block.Transactions {
					fileLogger.Printf("\nGIAO DỊCH TRONG BLOCK #%d:", block.Number)

					// Lấy thông tin giao dịch
					tx, err := getTransactionVechainWS(txID, veChainNodeRESTURL)
					if err != nil {
						// Thử node dự phòng nếu node chính không hoạt động
						alternativeNodeURL := "https://sync-mainnet.vechain.org"
						log.Printf("Thử lấy giao dịch từ node thay thế: %s", alternativeNodeURL)
						tx, err = getTransactionVechainWS(txID, alternativeNodeURL)
						if err != nil {
							log.Printf("Không thể lấy thông tin giao dịch: %v", err)
							continue
						}
					}

					// Xác định thời gian khối
					var txBlockTime string

					// Với mỗi clause, tạo một log giao dịch riêng biệt
					for j, clause := range tx.Clauses {
						txType := determineTransactionTypeVechainWS(clause.Data)
						methodID := ""
						if len(clause.Data) >= 10 {
							methodID = clause.Data[:10]
						}

						// Lấy tên phương thức và chữ ký đầy đủ
						methodName, methodSignature := getMethodNameVechainWS(methodID)

						// Chuyển đổi giá trị hex sang decimal
						valueDecimal := hexToDecimalValueVechainWS(clause.Value)

						// Tạo log giao dịch
						txLog := TransactionLogVechainWS{
							BlockHeight:     fmt.Sprintf("%d", block.Number),
							BlockHash:       block.ID,
							BlockTime:       txBlockTime,
							ChainID:         "vechain",
							TxHash:          txID,
							From:            tx.Origin,
							To:              clause.To,
							Value:           clause.Value,
							ValueDecimal:    valueDecimal,
							Gas:             fmt.Sprintf("%d", tx.Gas),
							GasPriceCoef:    fmt.Sprintf("%d", tx.GasPriceCoef),
							Size:            fmt.Sprintf("%d", tx.Size),
							TxType:          txType,
							Method:          methodName,
							MethodSignature: methodSignature,
						}

						if tx.Nonce != "" {
							txLog.Nonce = tx.Nonce
						}

						// Định dạng JSON đẹp
						txJSON, _ := json.MarshalIndent(txLog, "", "    ")
						fileLogger.Println(string(txJSON))

						// Thêm log chi tiết cho clause nếu có data
						if len(clause.Data) > 10 && clause.Data != "0x" {
							clauseDetail := map[string]interface{}{
								"index":            j + 1,
								"method_id":        methodID,
								"method_name":      methodName,
								"method_signature": methodSignature,
								"data_length":      len(clause.Data),
							}

							if len(clause.Data) > 100 {
								clauseDetail["data_preview"] = clause.Data[:100] + "... (còn nữa)"
							} else {
								clauseDetail["data"] = clause.Data
							}

							clauseJSON, _ := json.MarshalIndent(clauseDetail, "", "    ")
							fileLogger.Printf("CHI TIẾT CLAUSE #%d:\n%s", j+1, string(clauseJSON))
						}
					}
				}
			} else {
				fileLogger.Printf("\nKhối này không chứa giao dịch nào")
			}

			// Dòng phân cách giữa các khối
			fileLogger.Println("\n" + strings.Repeat("-", 80) + "\n")
		}
	}()

	// Vòng lặp chính
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// Gửi ping để giữ kết nối
			if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("Lỗi gửi ping: ", err)
				return
			}
		case <-interrupt:
			log.Println("Nhận tín hiệu ngắt, đang đóng kết nối...")
			fileLogger.Println("=== Kết thúc ghi log ===")

			// Đóng kết nối
			err := c.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Lỗi gửi thông báo đóng: ", err)
			}

			// Đợi đóng kết nối
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
