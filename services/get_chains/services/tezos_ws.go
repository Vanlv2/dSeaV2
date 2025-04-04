package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Config cấu trúc cho cấu hình RPC
type Config struct {
	RPCURL string `json:"rpc"`
	Chain  string `json:"chain"`
}

// TransactionRecord chứa thông tin của giao dịch
type TransactionRecord struct {
	Address         string    `json:"address"`
	Amount          string    `json:"amount"`
	BlockNumber     int       `json:"block_number"`
	EventSignature  string    `json:"event_signature"`
	FromAddress     string    `json:"from_address"`
	LogIndex        int       `json:"log_index"`
	NameChain       string    `json:"name_chain"`
	RawData         string    `json:"raw_data"`
	Timestamp       string    `json:"timestamp"`
	ToAddress       string    `json:"to_address"`
	TransactionType string    `json:"transaction_type"`
	TxHash          string    `json:"tx_hash"`
	BlockTime       time.Time `json:"block_time,omitempty"` // Thêm trường để lưu thời gian block
}

// Block từ RPC
type Block struct {
	Hash   string `json:"hash"`
	Header struct {
		Level     int       `json:"level"`
		Timestamp time.Time `json:"timestamp"`
	} `json:"header"`
	Operations [4][]Operation `json:"operations"`
}

// Operation từ RPC
type Operation struct {
	Hash     string `json:"hash"`
	Contents []struct {
		Kind        string `json:"kind"`
		Source      string `json:"source,omitempty"`
		Destination string `json:"destination,omitempty"`
		Amount      string `json:"amount,omitempty"`
		EventSig    string `json:"event_signature,omitempty"`
		LogIndex    int    `json:"log_index,omitempty"`
		RawData     string `json:"raw_data,omitempty"`
	} `json:"contents"`
}

var logMutex sync.Mutex

// Handle_tezos_ws là hàm chính để xử lý Tezos qua giám sát realtime
func Handle_tezos_ws() {
	// Tạo thư mục log nếu chưa tồn tại
	if _, err := os.Stat("./log"); os.IsNotExist(err) {
		os.Mkdir("./log", 0755)
	}

	// Mở file log
	logFile, err := os.OpenFile("./services/get_chains/log/tezos_ws.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Không thể mở file log: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Đọc file cấu hình
	configFile, err := os.ReadFile("./services/get_chains/configs/config-tezos.json")
	if err != nil {
		log.Fatalf("Không thể đọc file cấu hình: %v", err)
	}

	var config Config
	if err := json.Unmarshal(configFile, &config); err != nil {
		log.Fatalf("Không thể parse file cấu hình: %v", err)
	}

	log.Printf("Bắt đầu giám sát blockchain %s qua RPC: %s\n", config.Chain, config.RPCURL)

	// Xử lý tín hiệu thoát
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Nhận tín hiệu thoát, đang dừng chương trình...")

		// Khởi tạo bộ đếm thời gian 30 giây
		timeout := time.After(30 * time.Second)

		// Hủy ngữ cảnh (context) để ngừng các hoạt động liên quan
		cancel()

		select {
		case <-timeout:
			log.Println("Chương trình đã thoát sau 30 giây.")
			os.Exit(0)
		case <-ctx.Done():
			log.Println("Chương trình đã thoát ngay lập tức.")
		}
	}()

	// Poll block mới mỗi 10 giây
	lastHeight := 0
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Chương trình đã kết thúc")
			return
		case <-ticker.C:
			latestBlock, err := getLatestBlock(config.RPCURL)
			if err != nil {
				log.Printf("Lỗi khi lấy block mới: %v", err)
				continue
			}

			latestHeight := latestBlock.Header.Level

			if lastHeight == 0 {
				lastHeight = latestHeight
				log.Printf("Khởi tạo với block #%d", lastHeight)
				processBlock(config.RPCURL, latestBlock, config.Chain)
			} else if latestHeight > lastHeight {
				maxMissed := 10
				for height := lastHeight + 1; height <= latestHeight && height <= lastHeight+maxMissed; height++ {
					missedBlock, err := getBlockByHeight(config.RPCURL, height)
					if err != nil {
						log.Printf("Lỗi khi lấy block #%d: %v", height, err)
						continue
					}
					log.Printf("Xử lý block #%d (bị bỏ sót hoặc mới)", height)
					processBlock(config.RPCURL, missedBlock, config.Chain)
				}
				if latestHeight > lastHeight+maxMissed {
					log.Printf("Bỏ qua các block từ #%d đến #%d", lastHeight+maxMissed+1, latestHeight)
					lastHeight = latestHeight
				} else {
					lastHeight = latestHeight
				}
			}
		}
	}
}

func getLatestBlock(rpcURL string) (Block, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	for retries := 0; retries < 3; retries++ {
		resp, err := client.Get(fmt.Sprintf("%s/chains/main/blocks/head", rpcURL))
		if err != nil {
			log.Printf("Lỗi HTTP khi lấy block mới (thử %d/3): %v", retries+1, err)
			time.Sleep(time.Second * time.Duration(retries+1))
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return Block{}, fmt.Errorf("RPC trả về mã lỗi: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return Block{}, err
		}

		var block Block
		if err := json.Unmarshal(body, &block); err != nil {
			return Block{}, err
		}
		return block, nil
	}
	return Block{}, fmt.Errorf("hết lần thử lại khi lấy block mới")
}

func getBlockByHeight(rpcURL string, height int) (Block, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	for retries := 0; retries < 3; retries++ {
		resp, err := client.Get(fmt.Sprintf("%s/chains/main/blocks/%d", rpcURL, height))
		if err != nil {
			time.Sleep(time.Second * time.Duration(retries+1))
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return Block{}, fmt.Errorf("RPC trả về mã lỗi: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return Block{}, err
		}

		var block Block
		if err := json.Unmarshal(body, &block); err != nil {
			return Block{}, err
		}
		return block, nil
	}
	return Block{}, fmt.Errorf("hết lần thử lại khi lấy block #%d", height)
}

// processBlock xử lý block và ghi log giao dịch
func processBlock(rpcURL string, block Block, chainName string) {
	var transactions []TransactionRecord

	for _, opGroup := range block.Operations[3] {
		for _, content := range opGroup.Contents {
			if content.Kind == "transaction" {
				// Tạo raw data từ nội dung giao dịch
				contentJSON, err := json.Marshal(content)
				if err != nil {
					log.Printf("Lỗi khi chuyển đổi nội dung giao dịch sang JSON: %v", err)
					continue
				}
				rawData := base64.StdEncoding.EncodeToString(contentJSON)

				// Tạo event signature
				eventSignature := fmt.Sprintf("%s.XTZ", content.Kind)

				record := TransactionRecord{
					Address:         content.Destination,
					Amount:          content.Amount,
					BlockNumber:     block.Header.Level,
					EventSignature:  eventSignature,
					FromAddress:     content.Source,
					LogIndex:        content.LogIndex,
					NameChain:       chainName,
					RawData:         rawData,
					Timestamp:       block.Header.Timestamp.Format(time.RFC3339),
					ToAddress:       content.Destination,
					TransactionType: content.Kind,
					TxHash:          opGroup.Hash,
					BlockTime:       block.Header.Timestamp,
				}
				transactions = append(transactions, record)
			}
		}
	}

	logMutex.Lock()
	defer logMutex.Unlock()
	log.Printf("BLOCK #%d | Hash: %s | Time: %s | Giao dịch: %d",
		block.Header.Level, block.Hash, block.Header.Timestamp, len(transactions))

	if len(transactions) > 0 {
		txJSON, _ := json.MarshalIndent(transactions, "", "  ")
		log.Printf("GIAO DỊCH TRONG BLOCK #%d:\n%s", block.Header.Level, string(txJSON))
	}
}
