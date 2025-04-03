package get_chains

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Cấu trúc dữ liệu cho BlockVechainHttp
type BlockVechainHttp struct {
	ID           string   `json:"id"`
	Number       uint64   `json:"number"`
	ParentID     string   `json:"parentID"`
	Timestamp    uint64   `json:"timestamp"`
	GasLimit     uint64   `json:"gasLimit"`
	GasUsed      uint64   `json:"gasUsed"`
	TotalScore   uint64   `json:"totalScore"`
	Beneficiary  string   `json:"beneficiary"`
	Size         uint64   `json:"size"`
	Transactions []string `json:"transactions"`
}

// Cấu trúc dữ liệu cho TransactionVechainHttp
type TransactionVechainHttp struct {
	ID           string `json:"id"`
	Origin       string `json:"origin"`
	Gas          uint64 `json:"gas"`
	GasPriceCoef uint64 `json:"gasPriceCoef"`
	Size         uint64 `json:"size"`
	Nonce        string `json:"nonce"`
	Clauses      []struct {
		To    string `json:"to"`
		Value string `json:"value"`
		Data  string `json:"data"`
	} `json:"clauses"`
	Meta *struct {
		BlockID        string `json:"blockID"`
		BlockNumber    uint64 `json:"blockNumber"`
		BlockTimestamp uint64 `json:"blockTimestamp"`
	} `json:"meta,omitempty"`
}

// Cấu trúc dữ liệu cho TransactionLoghttp
type TransactionLoghttp struct {
	BlockHeight     string `json:"block_height"`
	BlockHash       string `json:"block_hash"`
	BlockTime       string `json:"block_time"`
	ChainID         string `json:"chain_id"`
	TxHash          string `json:"tx_hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	ValueDecimal    string `json:"value_decimal"`
	Gas             string `json:"gas"`
	GasPriceCoef    string `json:"gas_price_coef"`
	Size            string `json:"size"`
	TxType          string `json:"tx_type"`
	Method          string `json:"method"`
	MethodSignature string `json:"method_signature"`
	Nonce           string `json:"nonce"`
}

// Hàm chính để quét các khối theo khoảng thời gian
func scanBlocksByTimeRange(duration time.Duration) error {
	// Node chính và dự phòng
	primaryNodeURL := "https://mainnet.veblocks.net"
	backupNodeURL := "https://sync-mainnet.vechain.org"

	// Lấy khối mới nhất
	latestBlock, err := getLatestBlock(primaryNodeURL)
	if err != nil {
		log.Printf("Không thể lấy khối mới nhất từ node chính, thử node dự phòng: %v", err)
		latestBlock, err = getLatestBlock(backupNodeURL)
		if err != nil {
			return fmt.Errorf("không thể lấy khối mới nhất: %v", err)
		}
	}

	log.Printf("Khối mới nhất: #%d", latestBlock.Number)
	fileLogger.Printf("===== BẮT ĐẦU QUÉT TỪ KHỐI #%d =====", latestBlock.Number)

	// Tính thời gian tối thiểu cần quét đến
	currentTime := time.Now()
	minTimeToScan := currentTime.Add(-duration)

	log.Printf("Quét từ: %s đến: %s", currentTime.Format(time.RFC3339), minTimeToScan.Format(time.RFC3339))
	fileLogger.Printf("Quét từ: %s đến: %s", currentTime.Format(time.RFC3339), minTimeToScan.Format(time.RFC3339))

	// Bắt đầu quét từ khối mới nhất và lùi dần
	currentBlock := latestBlock
	processedBlocks := 0

	for {
		// Kiểm tra điều kiện dừng theo thời gian
		blockTime := time.Unix(int64(currentBlock.Timestamp), 0)
		if blockTime.Before(minTimeToScan) {
			log.Printf("Đã đạt đến khối có thời gian trước %s, kết thúc quét", minTimeToScan.Format(time.RFC3339))
			fileLogger.Printf("Đã đạt đến khối có thời gian trước %s, kết thúc quét", minTimeToScan.Format(time.RFC3339))
			break
		}

		// Xử lý khối hiện tại
		processBlockVechain(currentBlock, primaryNodeURL, backupNodeURL)
		processedBlocks++

		// Nếu đã quét quá nhiều khối, có thể cân nhắc dừng để tránh quá tải
		if processedBlocks > 10000 {
			log.Printf("Đã quét %d khối, đạt giới hạn an toàn", processedBlocks)
			fileLogger.Printf("Đã quét %d khối, đạt giới hạn an toàn", processedBlocks)
			break
		}

		// Lấy khối cha (khối trước đó)
		parentBlock, err := getBlockByID(currentBlock.ParentID, primaryNodeURL)
		if err != nil {
			log.Printf("Không thể lấy khối cha từ node chính, thử node dự phòng: %v", err)
			parentBlock, err = getBlockByID(currentBlock.ParentID, backupNodeURL)
			if err != nil {
				return fmt.Errorf("không thể lấy khối cha ID %s: %v", currentBlock.ParentID, err)
			}
		}

		// Cập nhật khối hiện tại là khối cha để tiếp tục quét ngược
		currentBlock = parentBlock

		// Thêm độ trễ nhỏ để tránh quá tải API
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Hoàn thành quét %d khối", processedBlocks)
	fileLogger.Printf("===== KẾT THÚC QUÉT - ĐÃ XỬ LÝ %d KHỐI =====", processedBlocks)
	return nil
}

// Lấy khối mới nhất
func getLatestBlock(nodeURL string) (*BlockVechainHttp, error) {
	apiURL := fmt.Sprintf("%s/blocks/best", nodeURL)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API trả về mã lỗi: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var block BlockVechainHttp
	if err := json.Unmarshal(bodyBytes, &block); err != nil {
		return nil, err
	}

	return &block, nil
}

// Lấy khối theo ID
func getBlockByID(blockID string, nodeURL string) (*BlockVechainHttp, error) {
	apiURL := fmt.Sprintf("%s/blocks/%s", nodeURL, blockID)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API trả về mã lỗi: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var block BlockVechainHttp
	if err := json.Unmarshal(bodyBytes, &block); err != nil {
		return nil, err
	}

	return &block, nil
}

// Lấy thông tin giao dịch
func getTransactionhttp(txID string, nodeURL string) (*TransactionVechainHttp, error) {
	apiURL := fmt.Sprintf("%s/transactions/%s", nodeURL, txID)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API trả về mã lỗi: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tx TransactionVechainHttp
	if err := json.Unmarshal(bodyBytes, &tx); err != nil {
		return nil, err
	}

	return &tx, nil
}

// Xác định loại giao dịch dựa trên dữ liệu
func determineTransactionTypehttp(data string) string {
	if data == "0x" || len(data) < 10 {
		return "Transfer"
	}
	return "Contract"
}

// Lấy tên phương thức và chữ ký đầy đủ
func getMethodNamehttp(methodID string) (string, string) {
	// Đây là một ánh xạ đơn giản, trong thực tế bạn có thể cần một cơ sở dữ liệu hoặc API để tra cứu
	methodMap := map[string]struct {
		Name      string
		Signature string
	}{
		"0xa9059cbb": {"transfer", "transfer(address,uint256)"},
		"0x23b872dd": {"transferFrom", "transferFrom(address,address,uint256)"},
		"0x095ea7b3": {"approve", "approve(address,uint256)"},
		"0x70a08231": {"balanceOf", "balanceOf(address)"},
		"0x18160ddd": {"totalSupply", "totalSupply()"},
		"0x313ce567": {"decimals", "decimals()"},
		"0x06fdde03": {"name", "name()"},
		"0x95d89b41": {"symbol", "symbol()"},
	}

	if method, ok := methodMap[methodID]; ok {
		return method.Name, method.Signature
	}
	return "Unknown", "Unknown"
}

// Chuyển đổi giá trị hex sang decimal
func hexToDecimalValuehttp(hexValue string) string {
	if hexValue == "0x" || hexValue == "" {
		return "0"
	}

	// Loại bỏ tiền tố 0x
	hexValue = strings.TrimPrefix(hexValue, "0x")

	// Chuyển đổi sang decimal
	var decimal uint64
	_, err := fmt.Sscanf(hexValue, "%x", &decimal)
	if err != nil {
		return "0"
	}

	return fmt.Sprintf("%d", decimal)
}

// Xử lý thông tin khối và các giao dịch trong khối
func processBlockVechain(block *BlockVechainHttp, primaryNodeURL, backupNodeURL string) {
	txCount := len(block.Transactions)
	blockTime := time.Unix(int64(block.Timestamp), 0).UTC()
	formattedBlockTime := blockTime.Format(time.RFC3339Nano)

	log.Printf("Đang xử lý khối #%d, ID: %s, có %d giao dịch, thời gian: %s",
		block.Number, block.ID, txCount, formattedBlockTime)

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

			// Lấy thông tin giao dịch từ node chính
			tx, err := getTransactionhttp(txID, primaryNodeURL)
			if err != nil {
				// Thử node dự phòng nếu node chính không hoạt động
				log.Printf("Thử lấy giao dịch từ node thay thế: %s", backupNodeURL)
				tx, err = getTransactionhttp(txID, backupNodeURL)
				if err != nil {
					log.Printf("Không thể lấy thông tin giao dịch: %v", err)
					continue
				}
			}

			// Xác định thời gian khối
			var txBlockTime string
			if tx.Meta != nil {
				txBlockTime = time.Unix(int64(tx.Meta.BlockTimestamp), 0).UTC().Format(time.RFC3339Nano)
			} else {
				txBlockTime = formattedBlockTime
			}

			// Với mỗi clause, tạo một log giao dịch riêng biệt
			for j, clause := range tx.Clauses {
				txType := determineTransactionTypehttp(clause.Data)
				methodID := ""
				if len(clause.Data) >= 10 {
					methodID = clause.Data[:10]
				}

				// Lấy tên phương thức và chữ ký đầy đủ
				methodName, methodSignature := getMethodNamehttp(methodID)

				// Chuyển đổi giá trị hex sang decimal
				valueDecimal := hexToDecimalValuehttp(clause.Value)

				// Tạo log giao dịch
				txLog := TransactionLoghttp{
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

// Hàm main để chạy chức năng quét
func handle_http_vechain() {
	// Quét các khối trong vòng 1 giờ trước
	duration := 1 * time.Hour

	log.Printf("Bắt đầu quét các khối VeChain trong khoảng %s vừa qua", duration)

	err := scanBlocksByTimeRange(duration)
	if err != nil {
		log.Fatalf("Lỗi khi quét khối: %v", err)
	}

	log.Println("Quét khối hoàn tất")
}
