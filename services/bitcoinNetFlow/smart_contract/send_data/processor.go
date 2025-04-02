package smartcontract

import (
	"fmt"
	"time"

	config "main/config/bitcoinNetFlow"
	calculator "main/services/bitcoinNetFlow/caculator_datas"
	StructData "main/services/bitcoinNetFlow/smart_contract"
)

// ChooseTypeConfig chọn cấu hình contract dựa trên loại thời gian
func ChooseTypeConfig(timeType string) StructData.ConfigContract {
	switch timeType {
	case "daily":
		return config.SetConfigContractDaily()
	case "weekly":
		return config.SetConfigContractWeekly()
	case "monthly":
		return config.SetConfigContractMonthly()
	default:
		return config.SetConfigContractDaily()
	}
}

// ChosseTypeSend chọn loại gửi dữ liệu dựa trên khoảng thời gian
func ChosseTypeSend(timeType string, config StructData.ConfigContract, methodName string, params map[string]interface{}) {
	switch timeType {
	case "daily":
		CallContractMethodDaily(config, methodName, params)
	case "weekly":
		CallContractMethodWeekly(config, methodName, params)
	case "monthly":
		CallContractMethodMonthly(config, methodName, params)
	}
}

// SendDataToSMC gửi dữ liệu thời gian thực và lịch sử lên smart contract
func SendDataToSMC(realTimeData map[time.Time]calculator.RealTimeFlowData, historicalData map[time.Time]calculator.HistoricalFlowData, timeType string) {
	// Lấy cấu hình contract dựa trên loại thời gian
	contractConfig := ChooseTypeConfig(timeType)

	// Xử lý dữ liệu thời gian thực (trước đây là Binance)
	if len(realTimeData) > 0 {
		for timestamp, data := range realTimeData {
			// Chuyển đổi từ float sang uint64 cho smart contract
			incoming := fmt.Sprintf("%v", data.Incoming)
			outgoing := fmt.Sprintf("%v", data.Outgoing)
			balance := fmt.Sprintf("%v", data.Balance)

			// Chuẩn bị tham số
			params := map[string]interface{}{
				"timestamp":    uint64(timestamp.Unix()),
				"incoming":     incoming,
				"outgoing":     outgoing,
				"balance":      balance,
				"tokenSymbol":  "BTC",
				"exchangeName": "Bitcoin",
			}

			// Gọi phương thức với tham số timeType để xác định loại thời gian
			ChosseTypeSend(timeType, contractConfig, "recordFlow", params)

			// Chờ một chút giữa các giao dịch để tránh quá tải
			time.Sleep(2 * time.Second)
		}
	}

	// Xử lý dữ liệu lịch sử (trước đây là Kraken)
	if len(historicalData) > 0 {
		for timestamp, data := range historicalData {
			incoming := fmt.Sprintf("%v", data.Incoming)
			outgoing := fmt.Sprintf("%v", data.Outgoing)
			balance := fmt.Sprintf("%v", data.Balance)

			// Chuẩn bị tham số
			params := map[string]interface{}{
				"timestamp":    uint64(timestamp.Unix()),
				"incoming":     incoming,
				"outgoing":     outgoing,
				"balance":      balance,
				"tokenSymbol":  "BTC",
				"exchangeName": "Bitcoin",
			}

			// Gọi phương thức với tham số timeType để xác định loại thời gian
			ChosseTypeSend(timeType, contractConfig, "recordFlow", params)

			// Chờ một chút giữa các giao dịch để tránh quá tải
			time.Sleep(2 * time.Second)
		}
	}
}
