package fearGreedindex

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"time"
)

type FormDataAdjusted struct {
	Timestamp           *big.Int `json:"timestamp"`
	Value               *big.Int `json:"value"`
	ValueClassification string   `json:"value_classification"`
}

type FearGreedResponse struct {
	Data []struct {
		Value           string `json:"value"`
		ValueClass      string `json:"value_classification"`
		Timestamp       string `json:"timestamp"`
		TimeUntilUpdate string `json:"time_until_update"` // Chỉ có ở phần tử đầu tiên
	} `json:"data"`
	Status struct {
		Timestamp    string `json:"timestamp"`
		ErrorCode    int    `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	} `json:"status"`
}

func getFearGreedData() ([]FormDataAdjusted, int, error) {
	url := "https://api.alternative.me/fng/?limit=30"

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Lỗi khi tạo yêu cầu: %v", err)
		return nil, 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Lỗi khi gửi yêu cầu: %v", err)
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Yêu cầu thất bại với status: %d", resp.StatusCode)
		return nil, 0, fmt.Errorf("yêu cầu thất bại với status: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Lỗi khi đọc phản hồi: %v", err)
		return nil, 0, err
	}

	var fgResponse FearGreedResponse
	err = json.Unmarshal(body, &fgResponse)
	if err != nil {
		log.Printf("Lỗi khi phân tích JSON: %v", err)
		return nil, 0, err
	}

	// Chuyển đổi tất cả 30 ngày dữ liệu
	var formDataList []FormDataAdjusted
	for _, item := range fgResponse.Data {
		value := new(big.Int)
		_, ok := value.SetString(item.Value, 10)
		if !ok {
			log.Printf("Lỗi khi chuyển đổi giá trị: %v", item.Value)
			return nil, 0, fmt.Errorf("lỗi khi chuyển đổi giá trị: %v", item.Value)
		}

		timestamp := new(big.Int)
		_, ok = timestamp.SetString(item.Timestamp, 10)
		if !ok {
			log.Printf("Lỗi khi chuyển đổi timestamp: %v", item.Timestamp)
			return nil, 0, fmt.Errorf("lỗi khi chuyển đổi timestamp: %v", item.Timestamp)
		}

		formData := FormDataAdjusted{
			Timestamp:           timestamp,
			Value:               value,
			ValueClassification: item.ValueClass,
		}
		formDataList = append(formDataList, formData)
	}

	// Lấy timeUntilUpdate từ phần tử đầu tiên
	timeUntilUpdate, err := strconv.Atoi(fgResponse.Data[0].TimeUntilUpdate)
	if err != nil {
		log.Printf("Lỗi khi chuyển đổi TimeUntilUpdate: %v", err)
		return nil, 0, err
	}

	return formDataList, timeUntilUpdate, nil
}

func FearGreedindex() {
	// Lấy và log 30 ngày đầu tiên
	fmt.Println("Đang lấy dữ liệu Fear & Greed Index cho 30 ngày trước...")
	formDataList, timeUntilUpdate, err := getFearGreedData()
	if err != nil {
		log.Printf("Có lỗi xảy ra khi lấy dữ liệu ban đầu: %v", err)
		time.Sleep(10 * time.Minute)
		return
	}

	// Log 30 ngày dữ liệu
	for i, formData := range formDataList {
		// Chuyển timestamp sang định dạng thời gian dễ đọc
		t := time.Unix(formData.Timestamp.Int64(), 0)
		fmt.Printf("Ngày %d (%s):\n", 30-i, t.Format("2006-01-02"))
		fmt.Printf("Timestamp: %s\n", formData.Timestamp.String())
		fmt.Printf("Value: %s\n", formData.Value.String())
		fmt.Printf("Value Classification: %s\n", formData.ValueClassification)
		fmt.Println("-------------------")
		// Gửi dữ liệu đến SMC (giả sử ConnectToSMC đã được định nghĩa)
		ConnectToSMC(formData)
	}

	// Vòng lặp để log dữ liệu mới sau mỗi lần cập nhật
	for {
		fmt.Printf("Chờ %d giây để lấy dữ liệu ngày tiếp theo...\n", timeUntilUpdate)
		time.Sleep(time.Duration(timeUntilUpdate) * time.Second)

		fmt.Println("Đang lấy dữ liệu Fear & Greed Index mới nhất...")
		formDataList, timeUntilUpdate, err = getFearGreedData()
		if err != nil {
			log.Printf("Có lỗi xảy ra: %v", err)
			time.Sleep(10 * time.Minute)
			continue
		}

		// Chỉ log ngày mới nhất (index 0)
		latest := formDataList[0]
		t := time.Unix(latest.Timestamp.Int64(), 0)
		fmt.Printf("Dữ liệu mới nhất (%s):\n", t.Format("2006-01-02"))
		fmt.Printf("Timestamp: %s\n", latest.Timestamp.String())
		fmt.Printf("Value: %s\n", latest.Value.String())
		fmt.Printf("Value Classification: %s\n", latest.ValueClassification)
		fmt.Printf("Thời gian cập nhật tiếp theo: %d giây\n\n", timeUntilUpdate)
		// Gửi dữ liệu mới nhất đến SMC
		ConnectToSMC(latest)
	}
}
