package methods

import (
	"fmt"
	"math/big"
)

func PrepareMethodArgsWeekly(methodName string, params map[string]interface{}) ([]interface{}, error) {
	var methodArgs []interface{}

	switch methodName {
	case "recordFlow":
		// Kiểm tra và thêm các tham số cho hàm recordFlow
		if timestamp, ok := params["timestamp"].(uint64); ok {
			methodArgs = append(methodArgs, big.NewInt(int64(timestamp)))
		} else {
			return nil, fmt.Errorf("missing or invalid 'timestamp' parameter for recordFlow method")
		}

		if incoming, ok := params["incoming"].(string); ok {
			methodArgs = append(methodArgs, incoming)
		} else {
			return nil, fmt.Errorf("missing or invalid 'incoming' parameter for recordFlow method")
		}

		if outgoing, ok := params["outgoing"].(string); ok {
			methodArgs = append(methodArgs, outgoing)
		} else {
			return nil, fmt.Errorf("missing or invalid 'outgoing' parameter for recordFlow method")
		}

		if balance, ok := params["balance"].(string); ok {
			methodArgs = append(methodArgs, balance)
		} else {
			return nil, fmt.Errorf("missing or invalid 'balance' parameter for recordFlow method")
		}

		if tokenSymbol, ok := params["tokenSymbol"].(string); ok {
			methodArgs = append(methodArgs, tokenSymbol)
		} else {
			return nil, fmt.Errorf("missing or invalid 'tokenSymbol' parameter for recordFlow method")
		}

		if exchangeName, ok := params["exchangeName"].(string); ok {
			methodArgs = append(methodArgs, exchangeName)
		} else {
			return nil, fmt.Errorf("missing or invalid 'exchangeName' parameter for recordFlow method")
		}
	default:
		return nil, fmt.Errorf("unsupported method: %s", methodName)
	}

	return methodArgs, nil
}
