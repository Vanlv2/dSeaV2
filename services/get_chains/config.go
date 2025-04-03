package get_chains

import (
	"encoding/json"
	"log"
	"os"
)

func load_config(filePath string, chainName string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open config file: %v", err)
	}
	defer file.Close()

	chainData := InitChainData(chainName)

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&chainData.Config)
	if err != nil {
		log.Fatalf("Failed to decode config file: %v", err)
	}
}
