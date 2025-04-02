package bitcoinNetFlow

import "main/services/bitcoinNetFlow/processer"

func BitcoinNetFlow() {
	go processer.Handle_daily_SMC()
	// go processer.Handle_weekly_SMC()
	// go processer.Handle_monthly_SMC()
}
