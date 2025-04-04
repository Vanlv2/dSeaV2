package entities

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/emirpasic/gods/maps/treemap"
)

// Cấu trúc để lưu trữ thông tin holder và cập nhật theo thời gian thực
type HolderCache struct {
	mutex      sync.RWMutex
	holderData *treemap.Map // Sử dụng treemap thay vì map thông thường
	tokenInfo  TokenInfo
	lastUpdate time.Time
}

func Entities() {
	// --- Lấy API Key từ biến môi trường ---
	covalentApiKey := "cqt_rQPfJ7vGRjF6yqYwdmwfcPVJ4ByH"

	// --- Thay đổi thông tin token BSC ở đây ---
	// Địa chỉ contract của BTCB trên BSC
	tokenContractAddress := "0x0555E30da8f98308EdB960aa94C0Db47230d2B9c"
	// ------------------------------------------------------------------

	if tokenContractAddress == "" {
		log.Fatal("Bạn cần cung cấp địa chỉ contract của token trên BSC.")
		return
	}

	// Tạo channel để bắt tín hiệu Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Số lượng holder cần lấy
	numberOfHolders := 100

	// Khởi tạo cache để lưu trữ thông tin holder với treemap
	// Sử dụng hàm so sánh để sắp xếp theo số dư giảm dần
	holderCache := HolderCache{
		holderData: treemap.NewWith(func(a, b interface{}) int {
			// So sánh ngược để sắp xếp giảm dần theo số dư
			holderA := a.(HolderInfo)
			holderB := b.(HolderInfo)

			// So sánh ReadableValue (số dư)
			cmp := holderA.ReadableValue.Cmp(holderB.ReadableValue)
			if cmp == 0 {
				// Nếu số dư bằng nhau, sắp xếp theo địa chỉ
				return strings.Compare(holderA.Address, holderB.Address)
			}
			return -cmp // Đảo ngược kết quả để sắp xếp giảm dần
		}),
	}

	// Tạo một goroutine để cập nhật dữ liệu holder theo thời gian thực
	go func() {
		for {
			// Cập nhật dữ liệu holder
			updateHolderData(&holderCache, covalentApiKey, tokenContractAddress, numberOfHolders)

			// Đợi 30 giây trước khi cập nhật lại
			time.Sleep(30 * time.Second)
		}
	}()

	// Chạy vòng lặp vô hạn để hiển thị dữ liệu
	for {
		// Kiểm tra xem có tín hiệu dừng không
		select {
		case <-sigChan:
			fmt.Println("\nĐã nhận tín hiệu dừng. Chương trình kết thúc.")
			return
		default:
			// Tiếp tục thực thi
		}

		// Hiển thị dữ liệu holder từ cache
		displayHolderData(&holderCache)

		// Đợi 2 giây trước khi hiển thị lại
		time.Sleep(2 * time.Second)
	}
}

// Hàm cập nhật dữ liệu holder
func updateHolderData(cache *HolderCache, apiKey string, tokenAddress string, pageSize int) {
	fmt.Printf("\nĐang cập nhật dữ liệu top %d holders của token...\n", pageSize)

	tokenInfo, holders, err := getBSCTopTokenHolders(apiKey, tokenAddress, pageSize)
	if err != nil {
		log.Printf("Không thể lấy dữ liệu top holders: %v\n", err)
		return
	}

	// Cập nhật cache
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	cache.tokenInfo = tokenInfo
	cache.lastUpdate = time.Now()

	// Tạo treemap mới để cập nhật
	newHolderMap := treemap.NewWith(func(a, b interface{}) int {
		// So sánh ngược để sắp xếp giảm dần theo số dư
		holderA := a.(HolderInfo)
		holderB := b.(HolderInfo)

		// So sánh ReadableValue (số dư)
		cmp := holderA.ReadableValue.Cmp(holderB.ReadableValue)
		if cmp == 0 {
			// Nếu số dư bằng nhau, sắp xếp theo địa chỉ
			return strings.Compare(holderA.Address, holderB.Address)
		}
		return -cmp // Đảo ngược kết quả để sắp xếp giảm dần
	})

	// Cập nhật thông tin holder
	for _, holder := range holders {
		newHolderMap.Put(holder, holder)
	}

	cache.holderData = newHolderMap

	fmt.Printf("Đã cập nhật dữ liệu %d holders thành công.\n", len(holders))
}

// Hàm hiển thị dữ liệu holder từ cache
func displayHolderData(cache *HolderCache) {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	if cache.holderData == nil || cache.holderData.Size() == 0 {
		fmt.Println("Đang chờ dữ liệu holder...")
		return
	}

	fmt.Printf("\n\n========== PHÂN TÍCH TOP %d HOLDERS ==========\n\n", cache.holderData.Size())

	// Lấy giá hiện tại của token
	tokenId := getTokenIdForCoinGecko(cache.tokenInfo.Symbol)
	fmt.Printf("Đang lấy dữ liệu cho token: %s (ID: %s)\n", cache.tokenInfo.Symbol, tokenId)

	currentPrice, err := getCurrentPrice(tokenId)
	if err != nil {
		currentPrice = 0
		fmt.Printf("Không thể lấy giá hiện tại của %s, sẽ hiển thị giá trị USD là 0: %v\n", cache.tokenInfo.Symbol, err)
	} else {
		fmt.Printf("Giá hiện tại của %s: $%.4f\n", cache.tokenInfo.Symbol, currentPrice)
	}

	// Lấy tổng cung lưu hành
	circulatingSupply, err := getCirculatingSupply(tokenId)
	if err != nil {
		circulatingSupply = 0
		fmt.Printf("Không thể lấy tổng cung lưu hành của %s, sẽ không hiển thị tỷ lệ phần trăm: %v\n", cache.tokenInfo.Symbol, err)
	} else {
		fmt.Printf("Tổng cung lưu hành của %s: %.2f\n", cache.tokenInfo.Symbol, circulatingSupply)
	}

	// In thông tin về top holders
	fmt.Printf("\n=== TOP %d HOLDERS CỦA %s (%s) ===\n",
		cache.holderData.Size(),
		cache.tokenInfo.Name,
		cache.tokenInfo.Symbol,
	)
	fmt.Printf("Cập nhật lần cuối: %s\n", cache.lastUpdate.Format("15:04:05 02/01/2006"))
	fmt.Printf("%-4s | %-42s | %-15s | %-15s | %-10s\n",
		"STT", "ĐỊA CHỈ VÍ",
		fmt.Sprintf("SỐ DƯ (%s)", cache.tokenInfo.Symbol),
		"GIÁ TRỊ (USD)",
		"TỶ LỆ (%)",
	)
	fmt.Println(strings.Repeat("-", 100))

	totalBalance := new(big.Float)

	// Duyệt qua các holder đã được sắp xếp
	i := 1
	it := cache.holderData.Iterator()
	for it.Next() {
		if i > 100 { // Giới hạn hiển thị 100 holders
			break
		}

		holder := it.Value().(HolderInfo)

		// Tính giá trị USD
		usdValue := new(big.Float).Mul(holder.ReadableValue, big.NewFloat(currentPrice))

		// Format giá trị USD
		var usdValueStr string
		if currentPrice > 0 {
			usdValueStr = fmt.Sprintf("%.2f", usdValue)
		} else {
			usdValueStr = "N/A"
		}

		// Tính tỷ lệ phần trăm
		var percentageStr string
		if circulatingSupply > 0 {
			// Chuyển ReadableValue từ big.Float sang float64 để tính toán
			holderBalance, _ := holder.ReadableValue.Float64()
			// Không cần kiểm tra độ chính xác, chấp nhận một chút sai số
			percentage := (holderBalance / circulatingSupply) * 100
			percentageStr = fmt.Sprintf("%.4f", percentage)
		} else {
			percentageStr = "N/A"
		}

		// In thông tin holder
		fmt.Printf("%-4d | %-42s | %-15.8f | $%-15s | %-10s\n",
			i,
			holder.Address,
			holder.ReadableValue,
			usdValueStr,
			percentageStr,
		)

		// Cộng dồn tổng số dư
		totalBalance = new(big.Float).Add(totalBalance, holder.ReadableValue)

		i++
	}

	// Tính tổng giá trị USD
	totalUSD := new(big.Float).Mul(totalBalance, big.NewFloat(currentPrice))

	// Tính tỷ lệ phần trăm của tổng số dư
	var totalPercentageStr string
	if circulatingSupply > 0 {
		totalBalanceFloat, _ := totalBalance.Float64()
		totalPercentage := (totalBalanceFloat / circulatingSupply) * 100
		totalPercentageStr = fmt.Sprintf("%.4f", totalPercentage)
	} else {
		totalPercentageStr = "N/A"
	}

	// In tổng số dư và giá trị
	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("%-4s | %-42s | %-15.8f | $%-15.2f | %-10s\n",
		"",
		"TỔNG CỘNG",
		totalBalance,
		totalUSD,
		totalPercentageStr,
	)

	if circulatingSupply > 0 {
		fmt.Printf("\nTổng cung lưu hành của %s: %.2f\n", cache.tokenInfo.Symbol, circulatingSupply)

		// Tính tổng giá trị thị trường
		marketCap := circulatingSupply * currentPrice
		fmt.Printf("Tổng giá trị thị trường của %s: $%.2f\n", cache.tokenInfo.Symbol, marketCap)

		// Tính tỷ lệ nắm giữ của top holders
		totalBalanceFloat, _ := totalBalance.Float64()
		holdersPercentage := (totalBalanceFloat / circulatingSupply) * 100
		fmt.Printf("Top %d holders nắm giữ %.4f%% tổng cung lưu hành\n",
			cache.holderData.Size(), holdersPercentage)
	}
}

// Cấu trúc thông tin token
type TokenInfo struct {
	Name     string
	Symbol   string
	Decimals int
}

// Cấu trúc thông tin holder
type HolderInfo struct {
	Address       string
	RawBalance    string
	ReadableValue *big.Float
}

// Hàm lấy danh sách top holders của token trên BSC
func getBSCTopTokenHolders(apiKey, tokenAddress string, pageSize int) (TokenInfo, []HolderInfo, error) {
	// URL của Covalent API để lấy danh sách holders
	url := fmt.Sprintf("%s/%s/tokens/%s/token_holders/?key=%s&page-size=%d",
		covalentBaseURL, bscChainID, tokenAddress, apiKey, pageSize)

	// Gọi API
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return TokenInfo{}, nil, fmt.Errorf("lỗi khi gọi Covalent API: %w", err)
	}
	defer resp.Body.Close()

	// Đọc response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TokenInfo{}, nil, fmt.Errorf("lỗi khi đọc response body: %w", err)
	}

	// Parse JSON
	var response CovalentResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return TokenInfo{}, nil, fmt.Errorf("lỗi khi parse JSON: %w", err)
	}

	// Kiểm tra lỗi từ API
	if response.ErrorMessage != "" {
		return TokenInfo{}, nil, fmt.Errorf("covalent API trả về lỗi: %s", response.ErrorMessage)
	}

	// Kiểm tra xem có dữ liệu không
	if len(response.Data.Items) == 0 {
		return TokenInfo{}, nil, fmt.Errorf("không có dữ liệu holder")
	}

	// Lấy thông tin token từ item đầu tiên
	firstItem := response.Data.Items[0]
	tokenInfo := TokenInfo{
		Name:     firstItem.ContractName,
		Symbol:   firstItem.ContractTickerSymbol,
		Decimals: firstItem.ContractDecimals,
	}

	// Chuyển đổi dữ liệu holder
	holders := make([]HolderInfo, 0, len(response.Data.Items))
	for _, item := range response.Data.Items {
		// Chuyển đổi balance từ string sang số thực
		readableValue, err := convertBalance(item.Balance, item.ContractDecimals)
		if err != nil {
			log.Printf("Lỗi khi chuyển đổi balance cho địa chỉ %s: %v\n", item.Address, err)
			continue
		}

		holders = append(holders, HolderInfo{
			Address:       item.Address,
			RawBalance:    item.Balance,
			ReadableValue: readableValue,
		})
	}

	return tokenInfo, holders, nil
}

// Cấu trúc dữ liệu cho Covalent API response
type CovalentResponse struct {
	Data         CovalentData `json:"data"`
	ErrorMessage string       `json:"error_message"`
	ErrorCode    *int         `json:"error_code"` // Dùng con trỏ vì có thể là null
}
