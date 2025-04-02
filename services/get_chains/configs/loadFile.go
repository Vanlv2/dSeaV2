package configs

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func LoadConfigLang(filePath string) (*Config, error) {
	// Mở file config
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	// Khởi tạo biến để lưu cấu hình
	var config Config

	// Đọc và decode file JSON
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config file: %v", err)
	}

	// Normalize các địa chỉ contract thành chữ thường
	config.ETHContractAddress = strings.ToLower(config.ETHContractAddress)
	config.USDTContractAddress = strings.ToLower(config.USDTContractAddress)
	config.USDCContractAddress = strings.ToLower(config.USDCContractAddress)
	config.WrappedBTCAddress = strings.ToLower(config.WrappedBTCAddress)

	// Trả về cấu hình đã đọc
	return &config, nil
}
