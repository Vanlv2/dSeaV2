package ohlcv

func RunOHLCV() {
	go WebSocketOHLCVDay()
	go WebSocketOHLCVWeek()
	go WebSocketOHLCVMonth()
}
