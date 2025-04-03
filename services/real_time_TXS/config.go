package real_time_TXS

import (
	"encoding/json"
	"os"
)

func load_config(filePath string, chainName string) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	chainData := InitChainData(chainName)

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&chainData.Config)
	if err != nil {
		return
	}
}
