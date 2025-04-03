package configs

import (
	"encoding/json"
	"fmt"
	"os"

	"main/services/get_chains/model"
)

// LoadConfig tải cấu hình từ file JSON và lưu vào ChainDataMap
// Trong package config
func LoadConfig(filePath, chainName string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	data, exists := model.ChainDataMapVan[chainName]
	if !exists {
		if chainName == "tezos" {
			data = &model.ChainDataTezos{
				LastProcessedBlock: 0,
				LogData:            make(map[string]interface{}),
			}
		} else if chainName == "elrond" {
			data = &model.ChainDataElrond{
				LastProcessedBlock: 0,
				LogData:            make(map[string]interface{}),
			}
		} else if chainName == "algorand" {
			data = &model.ChainDataAlgorand{
				LastProcessedBlock: 0,
				LogData:            make(map[string]interface{}),
			}
		} else if chainName == "stellar" { // Thêm hỗ trợ cho stellar
			data = &model.ChainDataStellarVan{
				LastProcessedLedger: 0,
				LogData:             make(map[string]interface{}),
			}
		} else {
			return nil // Không hỗ trợ chuỗi này
		}
		model.ChainDataMapVan[chainName] = data
	}

	// Decode cấu hình dựa trên loại chuỗi
	if chainName == "tezos" {
		var config model.ConfigTezos
		if err := json.NewDecoder(file).Decode(&config); err != nil {
			return err
		}
		data.SetConfigVan(config)
	} else if chainName == "elrond" {
		var config model.ConfigElrond
		if err := json.NewDecoder(file).Decode(&config); err != nil {
			return err
		}
		data.SetConfigVan(config)
	} else if chainName == "algorand" {
		var config model.ConfigAlgorand
		if err := json.NewDecoder(file).Decode(&config); err != nil {
			return err
		}
		data.SetConfigVan(config)
	} else if chainName == "stellar" { // Thêm trường hợp stellar
		var config model.ConfigStellar
		if err := json.NewDecoder(file).Decode(&config); err != nil {
			return err
		}
		if config.HorizonURL == "" {
			return fmt.Errorf("horizon_url is empty in config file")
		}
		data.SetConfigVan(config)
	}

	return nil
}

// GetChainData lấy dữ liệu chuỗi từ ChainDataMap
func GetChainData(chainName string) model.ChainDataVan {
	data, exists := model.ChainDataMapVan[chainName]
	if !exists {
		return nil
	}
	return data
}
