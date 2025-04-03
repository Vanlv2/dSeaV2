package main

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// --- Cấu hình API ---
const (
	// API Key cho BscScan
	bscscanAPIKey = "TZR6PYQJPSREBUJHTYWP948TXHD3MXNQ7W"
)

func main() {
	// --- Lấy API Key từ biến môi trường ---
	covalentApiKey := "cqt_rQPfJ7vGRjF6yqYwdmwfcPVJ4ByH"

	// --- Thay đổi thông tin token BSC ở đây ---
	// Địa chỉ contract của WBTC trên BSC
	tokenContractAddress := "0x0555E30da8f98308EdB960aa94C0Db47230d2B9c"
	// ------------------------------------------------------------------

	if tokenContractAddress == "" {
		log.Fatal("Bạn cần cung cấp địa chỉ contract của token trên BSC.")
		return
	}

	// Tạo channel để bắt tín hiệu Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Biến đếm số holder
	numberOfHolders := 1

	// Chạy vòng lặp vô hạn cho đến khi nhận tín hiệu dừng
	for {
		// Kiểm tra xem có tín hiệu dừng không
		select {
		case <-sigChan:
			fmt.Println("\nĐã nhận tín hiệu dừng. Chương trình kết thúc.")
			return
		default:
			// Tiếp tục thực thi
		}

		fmt.Printf("\n\n========== PHÂN TÍCH TOP %d HOLDERS ==========\n\n", numberOfHolders)

		// Bước 1: Lấy danh sách top holder của token
		fmt.Printf("Đang lấy top %d holders của WBTC...\n", numberOfHolders)
		tokenInfo, holders, err := getBSCTopTokenHolders(covalentApiKey, tokenContractAddress, numberOfHolders)
		if err != nil {
			log.Printf("Không thể lấy dữ liệu top holders: %v\n", err)
			// Nếu có lỗi, đợi một chút rồi thử lại
			time.Sleep(5 * time.Second)
			continue
		}

		// Lấy giá hiện tại của WBTC
		currentPrice, err := getCurrentPrice("wrapped-bitcoin")
		if err != nil {
			currentPrice = 0
			fmt.Println("Không thể lấy giá hiện tại của WBTC, sẽ hiển thị giá trị USD là 0")
		}

		// In thông tin về top holders
		fmt.Printf("\n=== TOP %d HOLDERS CỦA %s (%s) ===\n",
			len(holders),
			tokenInfo.Name,
			tokenInfo.Symbol,
		)
		fmt.Printf("%-4s | %-42s | %-15s | %-15s\n", "STT", "ĐỊA CHỈ VÍ", "SỐ DƯ (WBTC)", "GIÁ TRỊ (USD)")
		fmt.Println(strings.Repeat("-", 85))

		totalBalance := new(big.Float) // Tính tổng balance để tham khảo
		totalUsdValue := 0.0

		for i, holder := range holders {
			var balanceStr string
			var usdValueStr string

			if holder.ReadableValue != nil {
				balanceStr = holder.ReadableValue.Text('f', 8)
				totalBalance.Add(totalBalance, holder.ReadableValue)

				// Tính giá trị USD
				balanceFloat, _ := holder.ReadableValue.Float64()
				usdValue := balanceFloat * currentPrice
				totalUsdValue += usdValue
				usdValueStr = fmt.Sprintf("%.2f", usdValue)
			} else {
				balanceStr = "N/A"
				usdValueStr = "N/A"
			}

			fmt.Printf("%-4d | %-42s | %-15s | $%-15s\n",
				i+1,
				holder.Address,
				balanceStr,
				usdValueStr,
			)
		}

		fmt.Println(strings.Repeat("-", 85))
		fmt.Printf("%-4s | %-42s | %-15s | $%-15.2f\n",
			"",
			"TỔNG CỘNG",
			totalBalance.Text('f', 8),
			totalUsdValue,
		)
		fmt.Println()

		// Bước 2: Phân tích giao dịch của từng holder
		fmt.Printf("=== PHÂN TÍCH GIAO DỊCH CỦA CÁC HOLDER ===\n\n")

		// Tạo slice để lưu tất cả giao dịch từ tất cả holder
		var allTransactions []ProcessedTransaction

		for i, holder := range holders {
			fmt.Printf("Đang phân tích holder %d/%d: %s\n", i+1, len(holders), holder.Address)

			// Lấy lịch sử giao dịch của holder
			stats, err := getTokenTransfers(bscscanAPIKey, holder.Address, tokenContractAddress)
			if err != nil {
				fmt.Printf("Lỗi khi lấy giao dịch: %v\n\n", err)
				continue
			}

			if stats.TotalTransactions == 0 {
				fmt.Printf("Không tìm thấy giao dịch nào\n\n")
				continue
			}

			// Thêm tất cả giao dịch vào slice tổng hợp
			allTransactions = append(allTransactions, stats.Transactions...)

			// In thông tin giao dịch của holder
			fmt.Printf("Tìm thấy %d giao dịch: %d deposit, %d withdrawal\n\n",
				stats.TotalTransactions,
				stats.DepositCount,
				stats.WithdrawalCount,
			)

			// In bảng giao dịch
			fmt.Printf("%-19s | %-10s | %-15s | %-15s | %-42s\n",
				"THỜI GIAN", "LOẠI GD", "SỐ LƯỢNG", "GIÁ TRỊ (USD)", "ĐỊA CHỈ ĐỐI TÁC")
			fmt.Println(strings.Repeat("-", 110))

			for _, tx := range stats.Transactions {
				timeStr := time.Unix(tx.Timestamp, 0).Format("2006-01-02 15:04:05")

				// Xác định địa chỉ đối tác (người gửi hoặc người nhận)
				var partnerAddress string
				if tx.TransactionType == "deposit" {
					partnerAddress = tx.FromAddress
				} else {
					partnerAddress = tx.ToAddress
				}

				fmt.Printf("%-19s | %-10s | %-15s | $%-15s | %-42s\n",
					timeStr,
					tx.TransactionType,
					tx.AssetAmount,
					tx.UsdValue,
					partnerAddress,
				)
			}

			fmt.Println()

			// Thêm delay nhỏ để tránh rate limit của API
			time.Sleep(1 * time.Second)
		}

		// Bước 3: Tổng hợp và phân tích tất cả giao dịch
		if len(allTransactions) > 0 {
			fmt.Printf("=== TỔNG HỢP GIAO DỊCH ===\n\n")

			// Đếm số lượng giao dịch theo loại
			depositCount := 0
			withdrawalCount := 0
			for _, tx := range allTransactions {
				if tx.TransactionType == "deposit" {
					depositCount++
				} else if tx.TransactionType == "withdrawal" {
					withdrawalCount++
				}
			}

			fmt.Printf("Tổng số giao dịch: %d (Deposit: %d, Withdrawal: %d)\n\n",
				len(allTransactions),
				depositCount,
				withdrawalCount,
			)

			// Tìm 5 giao dịch lớn nhất theo giá trị USD
			fmt.Printf("=== 5 GIAO DỊCH LỚN NHẤT ===\n")

			// Sắp xếp giao dịch theo giá trị USD (giảm dần)
			sort.Slice(allTransactions, func(i, j int) bool {
				// Parse giá trị USD từ string sang float
				usdI, errI := strconv.ParseFloat(allTransactions[i].UsdValue, 64)
				usdJ, errJ := strconv.ParseFloat(allTransactions[j].UsdValue, 64)

				// Nếu có lỗi khi parse, đặt giá trị là 0
				if errI != nil {
					usdI = 0
				}
				if errJ != nil {
					usdJ = 0
				}

				return usdI > usdJ
			})

			// Hiển thị 5 giao dịch lớn nhất
			fmt.Printf("%-19s | %-10s | %-15s | %-15s | %-42s\n",
				"THỜI GIAN", "LOẠI GD", "SỐ LƯỢNG", "GIÁ TRỊ (USD)", "ĐỊA CHỈ VÍ")
			fmt.Println(strings.Repeat("-", 110))

			maxTxToShow := 5
			if len(allTransactions) < maxTxToShow {
				maxTxToShow = len(allTransactions)
			}

			for i := 0; i < maxTxToShow; i++ {
				tx := allTransactions[i]
				timeStr := time.Unix(tx.Timestamp, 0).Format("2006-01-02 15:04:05")

				fmt.Printf("%-19s | %-10s | %-15s | $%-15s | %-42s\n",
					timeStr,
					tx.TransactionType,
					tx.AssetAmount,
					tx.UsdValue,
					tx.WalletAddress,
				)
			}

			fmt.Println("\nPhân tích hoàn tất cho top", numberOfHolders, "holders!")
		}

		// Tăng số lượng holder lên 1 cho lần chạy tiếp theo
		numberOfHolders++

		// Đợi một khoảng thời gian trước khi tiếp tục vòng lặp tiếp theo
		// để tránh rate limit của API và cho người dùng thời gian đọc kết quả
		fmt.Printf("\nĐợi 5 giây trước khi phân tích top %d holders...\n", numberOfHolders)
		time.Sleep(5 * time.Second)
	}
}
