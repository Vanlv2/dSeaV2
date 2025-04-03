package real_time_TXS

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var eventSignature = make(map[string][]string)

var alwaysCallApi = true

type SignatureResponse struct {
	Next     string      `json:"next"`
	Previous string      `json:"previous"`
	Count    int         `json:"count"`
	Results  []Signature `json:"results"`
}

type Signature struct {
	ID             int    `json:"id"`
	CreatedAt      string `json:"created_at"`
	TextSignature  string `json:"text_signature"`
	HexSignature   string `json:"hex_signature"`
	BytesSignature string `json:"bytes_signature"`
}

type OpenchainSignatureResponse struct {
	Ok     bool                     `json:"ok"`
	Result OpenchainSignatureResult `json:"result"`
}

type OpenchainSignatureResult struct {
	Event    map[string][]OpenchainSignatureItem `json:"event"`
	Function map[string][]OpenchainSignatureItem `json:"function"`
}

type OpenchainSignatureItem struct {
	Name     string `json:"name"`
	Filtered bool   `json:"filtered"`
}

var client = &http.Client{
	Timeout: 10 * time.Second,
}

func ExtractEventName(textSignature string) string {
	parenIndex := strings.Index(textSignature, "(")
	if parenIndex == -1 {
		return textSignature
	}

	return textSignature[:parenIndex]
}

func GetTextSignatureFromHex(hexSignature string) ([]string, error) {
	if !strings.HasPrefix(hexSignature, "0x") {
		hexSignature = "0x" + hexSignature
	}

	url := fmt.Sprintf("https://www.4byte.directory/api/v1/event-signatures/?hex_signature=%s", hexSignature)

	resp, err := client.Get(url)
	if err != nil {
		return GetTextSignatureFromOpenchain(hexSignature)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GetTextSignatureFromOpenchain(hexSignature)
	}

	var response SignatureResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return GetTextSignatureFromOpenchain(hexSignature)
	}

	if response.Count == 0 {
		return GetTextSignatureFromOpenchain(hexSignature)
	}

	var textSignatures []string
	for _, sig := range response.Results {
		textSignatures = append(textSignatures, sig.TextSignature)
	}

	return textSignatures, nil
}

func GetTextSignatureFromOpenchain(hexSignature string) ([]string, error) {
	if !strings.HasPrefix(hexSignature, "0x") {
		hexSignature = "0x" + hexSignature
	}

	url := fmt.Sprintf("https://api.openchain.xyz/signature-database/v1/lookup?event=%s", hexSignature)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi tạo request đến Openchain API: %v", err)
	}
	
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi gửi request đến Openchain API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi đọc response từ Openchain API: %v", err)
	}

	var response OpenchainSignatureResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi parse JSON từ Openchain API: %v", err)
	}

	if !response.Ok {
		return nil, fmt.Errorf("API trả về status không thành công")
	}

	var textSignatures []string
	
	if signatures, exists := response.Result.Event[hexSignature]; exists && len(signatures) > 0 {
		for _, sig := range signatures {
			textSignatures = append(textSignatures, sig.Name)
		}
	}

	if len(textSignatures) == 0 {
		return nil, fmt.Errorf("không tìm thấy text signature nào từ Openchain API")
	}

	return textSignatures, nil
}

func Parse_event_signature_name(hexSignature string) (string, error) {
	if !alwaysCallApi {
		if eventSignatureNames, exists := eventSignature[hexSignature]; exists && len(eventSignatureNames) > 0 {
			return eventSignatureNames[0], nil
		}
	}

	textSignatures, err := GetTextSignatureFromHex(hexSignature)
	if err != nil {
		if eventSignatureNames, exists := eventSignature[hexSignature]; exists && len(eventSignatureNames) > 0 {
			return eventSignatureNames[0], nil
		}
		return "Unknown", err
	}

	if len(textSignatures) > 0 {
		eventName := ExtractEventName(textSignatures[0])
		if eventSignature[hexSignature] == nil {
			eventSignature[hexSignature] = []string{eventName}
		} else {
			found := false
			for _, existingName := range eventSignature[hexSignature] {
				if existingName == eventName {
					found = true
					break
				}
			}
			if !found {
				eventSignature[hexSignature] = append(eventSignature[hexSignature], eventName)
			}
		}
		return eventName, nil
	}

	if eventSignatureNames, exists := eventSignature[hexSignature]; exists && len(eventSignatureNames) > 0 {
		return eventSignatureNames[0], nil
	}

	return "Unknown", fmt.Errorf("không tìm thấy text signature")
}
