package configs

type Config struct {
	RPC                 string `json:"rpc"`
	WssRPC              string `json:"wssRpc"`
	ETHContractAddress  string `json:"ethContractAddress"`
	USDTContractAddress string `json:"usdtContractAddress"`
	USDCContractAddress string `json:"usdcContractAddress"`
	WrappedBTCAddress   string `json:"wrappedBTCAddress"`
	TransferSignature   string `json:"transferSignature"`
	Chain               string `json:"chain"`
	TimeNeedToBlock     int    `json:"timeNeedToBlock"`
}
