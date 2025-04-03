package entities

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// Các hằng số và cấu trúc dữ liệu chung
const (
	BscScanAPIBaseURL     = "https://api.bscscan.com/api"
	BTCBTokenAddress      = "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c" // Địa chỉ token BTCB trên BSC
	BTCBDecimals          = 18
	MonitoringInterval    = 5 * time.Second
	ExchangeThreshold     = 1000.0 // 1000 BTCB cho sàn giao dịch
	OrganizationThreshold = 100.0  // 100 BTCB cho tổ chức
	CoinGeckoAPIURL       = "https://api.coingecko.com/api/v3/simple/price?ids=binance-bitcoin&vs_currencies=usd"
)

// Thông tin tổng cung BTCB
type BTCBSupplyInfo struct {
	TotalSupply float64
	LastUpdated time.Time
}

// Cấu trúc dữ liệu để lưu trữ thông tin địa chỉ
type AddressInfo struct {
	Balance    float64
	Percentage float64
}

// Cấu trúc dữ liệu theo dõi địa chỉ
type AddressMonitorData struct {
	Address      string
	Balance      float64
	USDValue     float64
	Percentage   float64
	LastUpdated  time.Time
	Transactions []TransactionInfo
	LastTxCount  int
}

// Thông tin giao dịch
type TransactionInfo struct {
	Hash      string
	Timestamp time.Time
	From      string
	To        string
	Value     float64
	IsInflow  bool
}

// Cấu trúc dữ liệu giao dịch từ BscScan
type Transaction struct {
	Hash        string `json:"hash"`
	TimeStamp   int64  `json:"timeStamp,string"`
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	TokenSymbol string `json:"tokenSymbol"`
}

// Cấu trúc phản hồi cho API số dư token
type TokenBalanceResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

// Cấu trúc phản hồi CoinGecko
type CoinGeckoResponse struct {
	BinanceBitcoin struct {
		USD float64 `json:"usd"`
	} `json:"binance-bitcoin"`
}

// Biến toàn cục
var (
	btcbSupplyInfo BTCBSupplyInfo
	apiKey         = "TZR6PYQJPSREBUJHTYWP948TXHD3MXNQ7W"
	stopChan       = make(chan struct{})
	wg             sync.WaitGroup

	// Danh sách các địa chỉ cần theo dõi
	TargetAddresses = []string{
		"0x882c173bc7ff3b7786ca16dfed3dfffb9ee7847b",
		// Thêm các địa chỉ khác vào đây
	}

	// Danh sách toàn bộ địa chỉ đã thu thập
	allAddresses []string

	// Map để lưu trữ dữ liệu theo dõi cho mỗi địa chỉ
	monitorDataMap = make(map[string]*AddressMonitorData)
	monitorMutex   sync.RWMutex

	// Mutex để bảo vệ truy cập đồng thời vào danh sách địa chỉ đang theo dõi
	monitoredAddressesMutex sync.RWMutex
	monitoredAddresses      = make(map[string]bool)

	// Khai báo các biến lưu trữ theo phân loại
	highBalanceAddresses  = make(map[string]AddressInfo) // Tất cả địa chỉ có số dư lớn
	organizationAddresses = make(map[string]AddressInfo) // Địa chỉ tổ chức (100-1000 BTCB)
	exchangeAddresses     = make(map[string]AddressInfo) // Địa chỉ sàn giao dịch (>1000 BTCB)

	// Lưu giá BTCB/USD để sử dụng trong chương trình
	currentBTCBPrice float64
)

// GetBTCBTotalSupply lấy tổng cung của BTCB từ API
func GetBTCBTotalSupply() (float64, error) {
	// Sử dụng API BSCScan để lấy tổng cung
	url := fmt.Sprintf("%s?module=stats&action=tokensupply&contractaddress=%s&apikey=%s",
		BscScanAPIBaseURL, BTCBTokenAddress, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("lỗi khi gọi API để lấy tổng cung: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("lỗi khi đọc dữ liệu từ API: %v", err)
	}

	var response struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("lỗi khi phân tích dữ liệu: %v", err)
	}

	if response.Status != "1" {
		return 0, fmt.Errorf("API trả về lỗi: %s", response.Message)
	}

	// Chuyển đổi từ wei sang BTCB
	totalSupply, err := convertWeiToToken(response.Result, BTCBDecimals)
	if err != nil {
		return 0, fmt.Errorf("lỗi khi chuyển đổi tổng cung: %v", err)
	}

	// Chuyển sang float
	totalSupplyFloat, err := strconv.ParseFloat(totalSupply, 64)
	if err != nil {
		return 0, fmt.Errorf("lỗi khi chuyển đổi tổng cung sang số: %v", err)
	}

	return totalSupplyFloat, nil
}

// Tính phần trăm nắm giữ so với tổng cung
func CalculateHoldingPercentage(balance float64) float64 {
	if btcbSupplyInfo.TotalSupply <= 0 {
		return 0
	}
	return (balance / btcbSupplyInfo.TotalSupply) * 100
}

// Cập nhật thông tin tổng cung BTCB định kỳ
func UpdateBTCBSupplyRegularly() {
	wg.Add(1)
	defer wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			supply, err := GetBTCBTotalSupply()
			if err != nil {
				log.Printf("[CẢNH BÁO] Không thể cập nhật thông tin tổng cung BTCB: %v", err)
			} else {
				btcbSupplyInfo.TotalSupply = supply
				btcbSupplyInfo.LastUpdated = time.Now()
				log.Printf("[TỔNG CUNG] BTCB: %.8f", supply)
			}
		}
	}
}

// IsAddressMonitored kiểm tra xem một địa chỉ đã được theo dõi chưa
func IsAddressMonitored(address string) bool {
	monitoredAddressesMutex.RLock()
	defer monitoredAddressesMutex.RUnlock()
	_, exists := monitoredAddresses[address]
	return exists
}

// AddAddressToMonitoring thêm địa chỉ vào danh sách theo dõi và tạo goroutine mới để theo dõi
func AddAddressToMonitoring(address string) {
	// Kiểm tra xem địa chỉ đã được theo dõi chưa
	if IsAddressMonitored(address) {
		return
	}

	// Thêm địa chỉ vào danh sách theo dõi
	monitoredAddressesMutex.Lock()
	monitoredAddresses[address] = true
	monitoredAddressesMutex.Unlock()

	// Tạo goroutine mới để theo dõi địa chỉ
	wg.Add(1)
	go func(addr string) {
		defer wg.Done()
		log.Printf("[THEO DÕI] Bắt đầu theo dõi địa chỉ: %s", addr)

		// Khởi tạo dữ liệu theo dõi
		monitorMutex.Lock()
		monitorDataMap[addr] = &AddressMonitorData{
			Address:     addr,
			LastUpdated: time.Now(),
		}
		monitorMutex.Unlock()

		// Cập nhật dữ liệu ban đầu
		updateAddressData(apiKey, addr)

		// Tạo ticker để cập nhật định kỳ
		ticker := time.NewTicker(MonitoringInterval)
		defer ticker.Stop()

		// Tạo ticker để kiểm tra giao dịch mới
		txTicker := time.NewTicker(15 * time.Second)
		defer txTicker.Stop()

		lastCheckTime := time.Now().Unix()

		for {
			select {
			case <-stopChan:
				log.Printf("[DỪNG] Dừng theo dõi địa chỉ: %s", addr)
				return
			case <-ticker.C:
				// Cập nhật dữ liệu định kỳ
				updateAddressData(apiKey, addr)
			case <-txTicker.C:
				// Kiểm tra giao dịch mới
				api := NewBscScanAPI(apiKey)
				transactions, err := api.GetAddressTransactions(addr, lastCheckTime)
				if err != nil {
					continue
				}

				if len(transactions) > 0 {
					log.Printf("[GIAO DỊCH] Phát hiện %d giao dịch mới cho địa chỉ %s", len(transactions), addr)
					lastCheckTime = time.Now().Unix()

					// Xử lý các giao dịch mới
					for _, tx := range transactions {
						txValueBTCB, _ := convertWeiToToken(tx.Value, BTCBDecimals)
						txValueBTCBFloat, _ := strconv.ParseFloat(txValueBTCB, 64)

						isInflow := tx.To == addr
						flowType := "NHẬN VÀO"
						if !isInflow {
							flowType = "GỬI RA"
						}

						fromTo := ""
						if isInflow {
							fromTo = "từ " + tx.From
						} else {
							fromTo = "đến " + tx.To
						}

						log.Printf("[GIAO DỊCH] %s [%s]: %s %s %.8f BTCB",
							addr,
							tx.Hash,
							flowType,
							fromTo,
							txValueBTCBFloat)

						// Cập nhật danh sách giao dịch
						timestamp := time.Unix(tx.TimeStamp, 0)

						monitorMutex.Lock()
						monitorDataMap[addr].Transactions = append(monitorDataMap[addr].Transactions, TransactionInfo{
							Hash:      tx.Hash,
							Timestamp: timestamp,
							From:      tx.From,
							To:        tx.To,
							Value:     txValueBTCBFloat,
							IsInflow:  isInflow,
						})

						// Giới hạn số lượng giao dịch lưu trữ (giữ 50 giao dịch gần nhất)
						if len(monitorDataMap[addr].Transactions) > 50 {
							// Cắt bỏ giao dịch cũ nhất
							monitorDataMap[addr].Transactions = monitorDataMap[addr].Transactions[len(monitorDataMap[addr].Transactions)-50:]
						}
						monitorMutex.Unlock()
					}

					// Cập nhật lại số dư sau khi có giao dịch mới
					updateAddressData(apiKey, addr)
				}
			}
		}
	}(address)
}

// ScanExchangeAddresses quét và lưu trữ các địa chỉ sàn giao dịch và tổ chức
func ScanExchangeAddresses() {
	wg.Add(1)
	defer wg.Done()

	log.Println("[BẮT ĐẦU] Quét các địa chỉ sàn giao dịch và tổ chức...")

	// Thu thập tất cả các địa chỉ ban đầu
	allAddresses, err := CollectAllAddresses(apiKey)
	if err != nil {
		log.Printf("[LỖI] Không thể thu thập địa chỉ: %v", err)
		return
	}

	// Quét các địa chỉ để tìm số dư lớn
	_, err = ScanAddressesForHighBalances(apiKey, allAddresses)
	if err != nil {
		log.Printf("[LỖI] Không thể quét số dư: %v", err)
	}

	// Lấy danh sách địa chỉ sàn giao dịch và thêm vào danh sách theo dõi
	for addr := range exchangeAddresses {
		log.Printf("[TÌM THẤY] Địa chỉ sàn giao dịch: %s với %.8f BTCB",
			addr, exchangeAddresses[addr].Balance)
		AddAddressToMonitoring(addr)
	}

	// Lấy danh sách địa chỉ tổ chức và thêm vào danh sách theo dõi
	for addr := range organizationAddresses {
		log.Printf("[TÌM THẤY] Địa chỉ tổ chức: %s với %.8f BTCB",
			addr, organizationAddresses[addr].Balance)
		AddAddressToMonitoring(addr)
	}

	// Tiếp tục quét định kỳ để tìm các địa chỉ mới
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			log.Println("[DỪNG] Dừng quét địa chỉ sàn giao dịch và tổ chức")
			return
		case <-ticker.C:
			// Thu thập các địa chỉ mới
			newAddresses, err := CollectAllAddresses(apiKey)
			if err != nil {
				continue
			}

			// Lấy một số địa chỉ ngẫu nhiên để quét
			randomAddresses := GetRandomAddresses(newAddresses, 100)

			// Quét các địa chỉ mới
			_, err = ScanAddressesForHighBalances(apiKey, randomAddresses)
			if err != nil {
				continue
			}

			// Kiểm tra các địa chỉ sàn giao dịch mới và thêm vào danh sách theo dõi
			for addr := range exchangeAddresses {
				if !IsAddressMonitored(addr) {
					log.Printf("[TÌM THẤY] Địa chỉ sàn giao dịch mới: %s với %.8f BTCB",
						addr, exchangeAddresses[addr].Balance)
					AddAddressToMonitoring(addr)
				}
			}

			// Kiểm tra các địa chỉ tổ chức mới và thêm vào danh sách theo dõi
			for addr := range organizationAddresses {
				if !IsAddressMonitored(addr) {
					log.Printf("[TÌM THẤY] Địa chỉ tổ chức mới: %s với %.8f BTCB",
						addr, organizationAddresses[addr].Balance)
					AddAddressToMonitoring(addr)
				}
			}
		}
	}
}

func start_follow() {
	// Cấu hình log để chỉ in ra terminal
	log.SetFlags(log.LstdFlags)

	log.Println("=== KHỞI ĐỘNG CHƯƠNG TRÌNH QUÉT VÀ THEO DÕI ĐỊA CHỈ BTCB ===")
	log.Printf("Ngưỡng số dư tối thiểu cho tổ chức: %.2f BTCB", OrganizationThreshold)
	log.Printf("Ngưỡng số dư tối thiểu cho sàn giao dịch: %.2f BTCB", ExchangeThreshold)

	// Khởi tạo map để lưu trữ dữ liệu theo dõi
	monitorDataMap = make(map[string]*AddressMonitorData)

	// Lấy tổng cung BTCB ban đầu
	totalSupply, err := GetBTCBTotalSupply()
	if err != nil {
		log.Printf("[CẢNH BÁO] Không thể lấy tổng cung BTCB ban đầu: %v", err)
	} else {
		btcbSupplyInfo.TotalSupply = totalSupply
		btcbSupplyInfo.LastUpdated = time.Now()
		log.Printf("[TỔNG CUNG] BTCB ban đầu: %.8f", totalSupply)
	}

	// Lấy giá BTCB/USD ban đầu
	currentBTCBPrice, err = GetBTCBPriceInUSD()
	if err != nil {
		log.Printf("[CẢNH BÁO] Không thể lấy giá BTCB/USD ban đầu: %v", err)
		currentBTCBPrice = 0 // Giá mặc định
	} else {
		log.Printf("[GIÁ] BTCB/USD ban đầu: %.2f USD", currentBTCBPrice)
	}

	// Bắt đầu các goroutine
	go UpdateBTCBSupplyRegularly()
	go UpdatePriceRegularly()

	// Bắt đầu quét các địa chỉ sàn giao dịch và tổ chức
	go ScanExchangeAddresses()

	// Xử lý tín hiệu để dừng chương trình một cách an toàn
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("[DỪNG] Nhận tín hiệu dừng chương trình. Đang dừng các goroutine...")

	// Đóng kênh stopChan để thông báo cho tất cả các goroutine dừng lại
	close(stopChan)

	// Đặt timeout để đảm bảo chương trình sẽ kết thúc ngay cả khi có goroutine bị treo
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Chờ tất cả các goroutine kết thúc hoặc timeout sau 5 giây
	select {
	case <-done:
		log.Println("[DỪNG] Tất cả các goroutine đã kết thúc an toàn.")
	case <-time.After(5 * time.Second):
		log.Println("[DỪNG] Timeout: Một số goroutine không kết thúc trong thời gian cho phép.")
	}

	log.Println("[DỪNG] Chương trình đã dừng.")
}
