package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type OKXConfig struct {
	APIKey     string `json:"api_key"`
	APISecret  string `json:"api_secret"`
	Passphrase string `json:"passphrase"`
	WSURL      string `json:"ws_url"`
}

type OKXTransaction struct {
	EventType       string
	TransactionType string
	Asset           string
	Amount          float64
	Timestamp       time.Time
	Collection      string
}
type LoginPayload struct {
	Op   string `json:"op"`
	Args []struct {
		APIKey     string `json:"apiKey"`
		Passphrase string `json:"passphrase"`
		Timestamp  string `json:"timestamp"`
		Sign       string `json:"sign"`
	} `json:"args"`
}

type SubscribePayload struct {
	Op   string           `json:"op"`
	Args []map[string]any `json:"args"`
}

type WebSocketResponse struct {
	Event  string         `json:"event"`
	Code   string         `json:"code"`
	Msg    string         `json:"msg"`
	Arg    map[string]any `json:"arg,omitempty"`
	ConnID string         `json:"connId,omitempty"`
	Data   []interface{}  `json:"data,omitempty"`
}

func saveOKXTransaction(db *sql.DB, tx OKXTransaction) error {
    if tx.Collection == "account_transactions" {
        query := `
            INSERT INTO account_transactions (timestamp, account_id, transaction_type, asset, amount, created_at)
            VALUES (?, ?, ?, ?, ?, NOW())
        `
        accountId := fmt.Sprintf("okx-%s-%d", tx.EventType, tx.Timestamp.UnixNano())
        _, err := db.Exec(query, tx.Timestamp, accountId, tx.TransactionType, tx.Asset, tx.Amount)
        return err
    }
    
    return fmt.Errorf("không tìm thấy bảng phù hợp cho collection: %s", tx.Collection)
}


// Utility functions
func safeString(obj map[string]any, key string, defaultValue string) string {
	if val, ok := obj[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func safeValue(obj map[string]any, key string, defaultValue interface{}) interface{} {
	if val, ok := obj[key]; ok {
		return val
	}
	return defaultValue
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

type OKXMessageHandler struct {
	logger *log.Logger
}

func NewOKXMessageHandler(logger *log.Logger) *OKXMessageHandler {
	return &OKXMessageHandler{logger: logger}
}

func (h *OKXMessageHandler) handleMessage(message map[string]any, db *sql.DB) {
	if h.handleErrorMessage(message) {
		return
	}

	// Handle pong messages
	if event, ok := message["event"].(string); ok && event == "ping" {
		h.handlePong(message)
		return
	}

	// Handle event-based messages
	event := safeString(message, "event", "")
	if event != "" {
		eventMethods := map[string]func(map[string]any){
			"login":              h.handleAuthenticate,
			"subscribe":          h.handleSubscriptionStatus,
			"channel-conn-count": h.handleChannelConnCount,
		}
		if method, ok := eventMethods[event]; ok {
			method(message)
		} else {
			h.logger.Printf("[OKX] Unhandled event: %s, message: %v", event, message)
		}
		return
	}

	// Handle channel-based messages
	arg, ok := safeValue(message, "arg", map[string]any{}).(map[string]any)
	if !ok {
		h.logger.Printf("[OKX] Failed to parse arg in message: %v", message)
		return
	}
	channel := safeString(arg, "channel", "")

	if channel != "" {
		transactionChannels := map[string]func(map[string]any, *sql.DB){
			"account":          h.handleAccountUpdate,
			"deposit-state":    h.handleDepositState,
			"withdrawal-state": h.handleWithdrawalState,
		}
		nonTransactionChannels := map[string]func(map[string]any){
			"bbo-tbt":              h.handleOrderBook,
			"books":                h.handleOrderBook,
			"books5":               h.handleOrderBook,
			"books50-l2-tbt":       h.handleOrderBook,
			"books-l2-tbt":         h.handleOrderBook,
			"tickers":              h.handleTicker,
			"trades":               h.handleTrades,
			"balance_and_position": h.handleBalanceAndPosition,
			"orders":               h.handleOrders,
			"orders-algo":          h.handleOrders,
		}

		if method, exists := transactionChannels[channel]; exists {
			method(message, db)
			return
		}
		if method, exists := nonTransactionChannels[channel]; exists {
			method(message)
			return
		}
		if len(channel) >= 6 && channel[:6] == "candle" {
			h.handleOHLCV(message)
			return
		}
		h.logger.Printf("[OKX] Unrecognized channel: %s, message: %v", channel, message)
		return
	}

	h.logger.Printf("[OKX] No event or channel found in message: %v", message)
}

func (h *OKXMessageHandler) handleErrorMessage(message map[string]any) bool {
	if event, ok := message["event"].(string); ok && event == "error" {
		code := safeString(message, "code", "")
		msg := safeString(message, "msg", "")
		h.logger.Printf("[OKX] Error received: Code=%s, Message=%s", code, msg)
		return true
	}
	return false
}

func (h *OKXMessageHandler) handlePong(message map[string]any) {
	h.logger.Printf("[OKX] Received pong, connection is alive")
}

func (h *OKXMessageHandler) handleAuthenticate(message map[string]any) {
	code := safeString(message, "code", "")
	if code == "0" {
		h.logger.Printf("[OKX] Authentication successful")
	} else {
		h.logger.Printf("[OKX] Authentication failed: %v", message)
	}
}

func (h *OKXMessageHandler) handleChannelConnCount(message map[string]any) {
	channel := safeString(message, "channel", "unknown")
	connCount := safeString(message, "connCount", "0")
	h.logger.Printf("[OKX] Channel connection count for %s: %s connections", channel, connCount)
}

func (h *OKXMessageHandler) handleSubscriptionStatus(message map[string]any) {
	arg, ok := safeValue(message, "arg", map[string]any{}).(map[string]any)
	if !ok {
		h.logger.Printf("[OKX] Failed to parse arg in subscription status: %v", message)
		return
	}
	channel := safeString(arg, "channel", "")
	h.logger.Printf("[OKX] Subscription status for channel %s: %v", channel, message)
}

func (h *OKXMessageHandler) handleOrderBook(message map[string]any) {
	arg, ok := safeValue(message, "arg", map[string]any{}).(map[string]any)
	if !ok {
		h.logger.Printf("[OKX] Failed to parse arg in order book update: %v", message)
		return
	}
	channel := safeString(arg, "channel", "")
	h.logger.Printf("[OKX] Order book update for channel %s: %v", channel, message)
}

func (h *OKXMessageHandler) handleTicker(message map[string]any) {
	h.logger.Printf("[OKX] Ticker update: %v", message)
}

func (h *OKXMessageHandler) handleTrades(message map[string]any) {
	h.logger.Printf("[OKX] Trade update: %v", message)
}

func (h *OKXMessageHandler) handleBalanceAndPosition(message map[string]any) {
	h.logger.Printf("[OKX] Balance and position update: %v", message)
}

func (h *OKXMessageHandler) handleOrders(message map[string]any) {
	h.logger.Printf("[OKX] Order update: %v", message)
}

func (h *OKXMessageHandler) handleOHLCV(message map[string]any) {
	h.logger.Printf("[OKX] OHLCV update: %v", message)
}

func (h *OKXMessageHandler) handleDepositState(message map[string]any, db *sql.DB) {
	dataList, ok := safeValue(message, "data", []interface{}{}).([]interface{})
	if !ok || len(dataList) == 0 {
		h.logger.Printf("[OKX] No data in deposit-state update: %v", message)
		return
	}

	for _, item := range dataList {
		detail, ok := item.(map[string]any)
		if !ok {
			h.logger.Printf("[OKX] Invalid deposit-state data format: %v", item)
			continue
		}

		asset := safeString(detail, "ccy", "")
		amountStr := safeString(detail, "amt", "0")
		amount := parseFloat(amountStr)
		if amount == 0 {
			continue
		}

		tsStr := safeString(detail, "ts", "0")
		tsMs := parseFloat(tsStr)
		timestamp := time.UnixMilli(int64(tsMs))

		transaction := OKXTransaction{
			EventType:       "deposit-state",
			TransactionType: "deposit",
			Asset:           asset,
			Amount:          amount,
			Timestamp:       timestamp,
			Collection:      "account_transactions",
		}

		if err := saveOKXTransaction(db, transaction); err != nil {
			h.logger.Printf("[OKX] Failed to save deposit transaction: %v", err)
		} else {
			h.logger.Printf("[OKX] Recorded deposit transaction: Asset=%s, Amount=%f, Timestamp=%s", asset, amount, timestamp)
		}
	}
}

func (h *OKXMessageHandler) handleWithdrawalState(message map[string]any, db *sql.DB) {
	dataList, ok := safeValue(message, "data", []interface{}{}).([]interface{})
	if !ok || len(dataList) == 0 {
		h.logger.Printf("[OKX] No data in withdrawal-state update: %v", message)
		return
	}

	for _, item := range dataList {
		detail, ok := item.(map[string]any)
		if !ok {
			h.logger.Printf("[OKX] Invalid withdrawal-state data format: %v", item)
			continue
		}

		asset := safeString(detail, "ccy", "")
		amountStr := safeString(detail, "amt", "0")
		amount := parseFloat(amountStr)
		if amount == 0 {
			continue
		}

		tsStr := safeString(detail, "ts", "0")
		tsMs := parseFloat(tsStr)
		timestamp := time.UnixMilli(int64(tsMs))

		transaction := OKXTransaction{
			EventType:       "withdrawal-state",
			TransactionType: "withdrawal",
			Asset:           asset,
			Amount:          amount,
			Timestamp:       timestamp,
			Collection:      "account_transactions",
		}

		if err := saveOKXTransaction(db, transaction); err != nil {
			h.logger.Printf("[OKX] Failed to save withdrawal transaction: %v", err)
		} else {
			h.logger.Printf("[OKX] Recorded withdrawal transaction: Asset=%s, Amount=%f, Timestamp=%s", asset, amount, timestamp)
		}
	}
}

func (h *OKXMessageHandler) handleAccountUpdate(message map[string]any, db *sql.DB) {
	dataList, ok := safeValue(message, "data", []interface{}{}).([]interface{})
	if !ok || len(dataList) == 0 {
		h.logger.Printf("[OKX] No data in account update: %v", message)
		return
	}

	for _, item := range dataList {
		details, ok := item.(map[string]any)
		if !ok {
			h.logger.Printf("[OKX] Invalid account data format: %v", item)
			continue
		}

		detailsList, ok := safeValue(details, "details", []interface{}{}).([]interface{})
		if !ok || len(detailsList) == 0 {
			continue
		}

		for _, detail := range detailsList {
			detailMap, ok := detail.(map[string]any)
			if !ok {
				continue
			}

			asset := safeString(detailMap, "ccy", "")
			balDeltaStr := safeString(detailMap, "balDelta", "0")
			amount := parseFloat(balDeltaStr)
			if amount == 0 {
				continue
			}

			transactionType := "deposit"
			if amount < 0 {
				transactionType = "withdrawal"
				amount = -amount // Convert to positive for consistency
			}

			tsStr := safeString(details, "uTime", "0")
			tsMs := parseFloat(tsStr)
			timestamp := time.UnixMilli(int64(tsMs))

			transaction := OKXTransaction{
				EventType:       "accountUpdate",
				TransactionType: transactionType,
				Asset:           asset,
				Amount:          amount,
				Timestamp:       timestamp,
				Collection:      "account_transactions",
			}

			if err := saveOKXTransaction(db, transaction); err != nil {
				h.logger.Printf("[OKX] Failed to save account update transaction: %v", err)
			} else {
				h.logger.Printf("[OKX] Recorded %s transaction: Asset=%s, Amount=%f, Timestamp=%s", transactionType, asset, amount, timestamp)
			}
		}
	}
}

func LoadOKXConfig(configFile string) (OKXConfig, error) {
	var cfg OKXConfig
	data, err := os.ReadFile(configFile)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.WSURL == "" {
		cfg.WSURL = "wss://ws.okx.com:8443/ws/v5/private" // Default URL if not provided
	}
	return cfg, nil
}

// Fetch server time for synchronization
func FetchOKXServerTime(logger *log.Logger) (int64, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://www.okx.com/api/v5/public/time")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			Ts string `json:"ts"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if result.Code != "0" || len(result.Data) == 0 {
		return 0, fmt.Errorf("failed to fetch server time: %s", result.Msg)
	}

	tsMs := parseFloat(result.Data[0].Ts)
	return int64(tsMs / 1000), nil // Convert milliseconds to seconds
}

func generateOKXSignature(apiKey, apiSecret, passphrase string, logger *log.Logger) (LoginPayload, error) {
	// Use Unix timestamp in seconds
	timestamp := fmt.Sprintf("%d", time.Now().UTC().Unix())
	signContent := timestamp + "login"
	logger.Printf("[OKX] Sign content for HMAC: %s", signContent)
	logger.Printf("[OKX] Using API Secret: %s", apiSecret) // Log the secret for debugging

	h := hmac.New(sha256.New, []byte(apiSecret))
	h.Write([]byte(signContent))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	payload := LoginPayload{
		Op: "login",
		Args: []struct {
			APIKey     string `json:"apiKey"`
			Passphrase string `json:"passphrase"`
			Timestamp  string `json:"timestamp"`
			Sign       string `json:"sign"`
		}{
			{
				APIKey:     apiKey,
				Passphrase: passphrase,
				Timestamp:  timestamp,
				Sign:       signature,
			},
		},
	}

	logger.Printf("[OKX] Generated login payload: %+v", payload)
	return payload, nil
}

func FetchDepositHistory(cfg OKXConfig, logger *log.Logger) ([]map[string]any, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://www.okx.com/api/v5/asset/deposit-history", nil)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	signContent := timestamp + "GET" + "/api/v5/asset/deposit-history"
	h := hmac.New(sha256.New, []byte(cfg.APISecret))
	h.Write([]byte(signContent))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	req.Header.Set("OK-ACCESS-KEY", cfg.APIKey)
	req.Header.Set("OK-ACCESS-SIGN", signature)
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", cfg.Passphrase)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Code string           `json:"code"`
		Msg  string           `json:"msg"`
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("error %s: %s", result.Code, result.Msg)
	}

	logger.Printf("[OKX] Fetched %d deposit records via REST API", len(result.Data))
	return result.Data, nil
}

func FetchWithdrawalHistory(cfg OKXConfig, logger *log.Logger) ([]map[string]any, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://www.okx.com/api/v5/asset/withdrawal-history", nil)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	signContent := timestamp + "GET" + "/api/v5/asset/withdrawal-history"
	h := hmac.New(sha256.New, []byte(cfg.APISecret))
	h.Write([]byte(signContent))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	req.Header.Set("OK-ACCESS-KEY", cfg.APIKey)
	req.Header.Set("OK-ACCESS-SIGN", signature)
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", cfg.Passphrase)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Code string           `json:"code"`
		Msg  string           `json:"msg"`
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Code != "0" {
		return nil, fmt.Errorf("error %s: %s", result.Code, result.Msg)
	}

	logger.Printf("[OKX] Fetched %d withdrawal records via REST API", len(result.Data))
	return result.Data, nil
}
func ProcessOKXAccountStream(cfg OKXConfig, db *sql.DB, stopChan <-chan struct{}, logger *log.Logger) {
	handler := NewOKXMessageHandler(logger)
	var mutex sync.Mutex
	var conn *websocket.Conn

	// Fetch server time to ensure clock synchronization
	serverTime, err := FetchOKXServerTime(logger)
	if err != nil {
		logger.Printf("[OKX] Failed to fetch server time: %v", err)
		logger.Printf("[OKX] Using local time, but this may cause timestamp errors")
	} else {
		localTime := time.Now().UTC().Unix()
		drift := serverTime - localTime
		if drift > 30 || drift < -30 {
			logger.Printf("[OKX] Clock drift detected: %d seconds. Consider synchronizing system time with NTP", drift)
		} else {
			logger.Printf("[OKX] Clock synchronized: drift is %d seconds", drift)
		}
	}

	// Log the loaded config for debugging
	logger.Printf("[OKX] Loaded config - API Key: %s, Passphrase: %s, WS URL: %s", cfg.APIKey, cfg.Passphrase, cfg.WSURL)

	reconnect := func() {
		mutex.Lock()
		defer mutex.Unlock()
		if conn != nil {
			conn.Close()
			conn = nil
		}
	}

	for {
		select {
		case <-stopChan:
			reconnect()
			return
		default:
			if conn == nil {
				dialer := &websocket.Dialer{
					HandshakeTimeout: 10 * time.Second,
				}
				var err error
				conn, _, err = dialer.Dial(cfg.WSURL, nil)
				if err != nil {
					logger.Printf("[OKX] WebSocket connection failed: %v", err)
					time.Sleep(5 * time.Second)
					continue
				}
				logger.Printf("[OKX] WebSocket connection established")
			}

			// Authenticate
			loginPayload, err := generateOKXSignature(cfg.APIKey, cfg.APISecret, cfg.Passphrase, logger)
			if err != nil {
				logger.Printf("[OKX] Failed to generate login signature: %v", err)
				reconnect()
				time.Sleep(5 * time.Second)
				continue
			}
			if err := conn.WriteJSON(loginPayload); err != nil {
				logger.Printf("[OKX] Failed to send login message: %v", err)
				reconnect()
				time.Sleep(5 * time.Second)
				continue
			}

			// Read login response
			mutex.Lock()
			_, loginResponse, err := conn.ReadMessage()
			mutex.Unlock()
			if err != nil {
				logger.Printf("[OKX] Failed to read login response: %v", err)
				reconnect()
				time.Sleep(5 * time.Second)
				continue
			}

			var loginResp map[string]any
			if err := json.Unmarshal(loginResponse, &loginResp); err != nil {
				logger.Printf("[OKX] Failed to parse login response: %v", err)
				reconnect()
				time.Sleep(5 * time.Second)
				continue
			}
			if code, ok := loginResp["code"].(string); !ok || code != "0" {
				logger.Printf("[OKX] Login failed: %v", string(loginResponse))
				reconnect()
				time.Sleep(5 * time.Second)
				continue
			}
			logger.Printf("[OKX] Login successful")

			// Subscribe to channels
			subscribePayload := SubscribePayload{
				Op: "subscribe",
				Args: []map[string]any{
					{"channel": "account"},
					{"channel": "balance_and_position"},
					{"channel": "deposit-state"},
					{"channel": "withdrawal-state"},
					{"channel": "tickers"},
					{"channel": "trades"},
					{"channel": "books"},
					{"channel": "orders"},
					{"channel": "orders-algo"},
				},
			}
			if err := conn.WriteJSON(subscribePayload); err != nil {
				logger.Printf("[OKX] Failed to subscribe to channels: %v", err)
				reconnect()
				time.Sleep(5 * time.Second)
				continue
			}
			logger.Printf("[OKX] Subscribed to channels")

			// Ping loop to keep connection alive
			pingDone := make(chan struct{})
			go func() {
				defer close(pingDone)
				ticker := time.NewTicker(10 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-stopChan:
						return
					case <-ticker.C:
						mutex.Lock()
						if conn != nil {
							if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
								logger.Printf("[OKX] Failed to send ping: %v", err)
							} else {
								logger.Printf("[OKX] Sent ping to keep connection alive")
							}
						}
						mutex.Unlock()
					}
				}
			}()

			// Pong handler for server-initiated pings
			pongDone := make(chan struct{})
			go func() {
				defer close(pongDone)
				for {
					select {
					case <-stopChan:
						return
					default:
						mutex.Lock()
						if conn == nil {
							mutex.Unlock()
							return
						}
						_, message, err := conn.ReadMessage()
						mutex.Unlock()
						if err != nil {
							logger.Printf("[OKX] Pong handler read error: %v", err)
							return
						}
						var msgMap map[string]any
						if err := json.Unmarshal(message, &msgMap); err == nil {
							if event, ok := msgMap["event"].(string); ok && event == "ping" {
								mutex.Lock()
								if err := conn.WriteJSON(map[string]any{"event": "pong"}); err != nil {
									logger.Printf("[OKX] Failed to send pong: %v", err)
								} else {
									logger.Printf("[OKX] Sent pong response")
								}
								mutex.Unlock()
							}
						} else if string(message) == "ping" {
							mutex.Lock()
							if err := conn.WriteMessage(websocket.TextMessage, []byte("pong")); err != nil {
								logger.Printf("[OKX] Failed to send pong: %v", err)
							} else {
								logger.Printf("[OKX] Sent pong response (text)")
							}
							mutex.Unlock()
						}
					}
				}
			}()

			// REST API fallback
			go func() {
				lastDepositTs := int64(0)
				lastWithdrawalTs := int64(0)
				ticker := time.NewTicker(30 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-stopChan:
						return
					case <-ticker.C:
						deposits, err := FetchDepositHistory(cfg, logger)
						if err != nil {
							logger.Printf("[OKX] Failed to fetch deposit history: %v", err)
						} else {
							for _, deposit := range deposits {
								tsStr := safeString(deposit, "ts", "0")
								tsMs := parseFloat(tsStr)
								if int64(tsMs) <= lastDepositTs {
									continue
								}
								lastDepositTs = int64(tsMs)
								asset := safeString(deposit, "ccy", "")
								amountStr := safeString(deposit, "amt", "0")
								amount := parseFloat(amountStr)
								timestamp := time.UnixMilli(int64(tsMs))

								transaction := OKXTransaction{
									EventType:       "deposit-state",
									TransactionType: "deposit",
									Asset:           asset,
									Amount:          amount,
									Timestamp:       timestamp,
									Collection:      "account_transactions",
								}
								if err := saveOKXTransaction(db, transaction); err != nil {
									logger.Printf("[OKX] Failed to save deposit transaction: %v", err)
								} else {
									logger.Printf("[OKX] Recorded deposit (REST API): %s, %f, %s", asset, amount, timestamp)
								}
							}
						}

						withdrawals, err := FetchWithdrawalHistory(cfg, logger)
						if err != nil {
							logger.Printf("[OKX] Failed to fetch withdrawal history: %v", err)
						} else {
							for _, withdrawal := range withdrawals {
								tsStr := safeString(withdrawal, "ts", "0")
								tsMs := parseFloat(tsStr)
								if int64(tsMs) <= lastWithdrawalTs {
									continue
								}
								lastWithdrawalTs = int64(tsMs)
								asset := safeString(withdrawal, "ccy", "")
								amountStr := safeString(withdrawal, "amt", "0")
								amount := parseFloat(amountStr)
								timestamp := time.UnixMilli(int64(tsMs))

								transaction := OKXTransaction{
									EventType:       "withdrawal-state",
									TransactionType: "withdrawal",
									Asset:           asset,
									Amount:          amount,
									Timestamp:       timestamp,
									Collection:      "account_transactions",
								}
								if err := saveOKXTransaction(db, transaction); err != nil {
									logger.Printf("[OKX] Failed to save withdrawal transaction: %v", err)
								} else {
									logger.Printf("[OKX] Recorded withdrawal (REST API): %s, %f, %s", asset, amount, timestamp)
								}
							}
						}
					}
				}
			}()

			// Message processing loop
			for {
				select {
				case <-stopChan:
					reconnect()
					<-pongDone
					<-pingDone
					return
				default:
					mutex.Lock()
					if conn == nil {
						mutex.Unlock()
						break
					}
					_, message, err := conn.ReadMessage()
					if err != nil {
						logger.Printf("[OKX] WebSocket read error: %v", err)
						reconnect()
						<-pongDone
						<-pingDone
						mutex.Unlock()
						break
					}
					mutex.Unlock()

					var msgMap map[string]any
					if err := json.Unmarshal(message, &msgMap); err != nil {
						logger.Printf("[OKX] Failed to parse message: %v", err)
						continue
					}
					logger.Printf("[OKX] Received message: %s", string(message))
					handler.handleMessage(msgMap, db)
				}
			}
		}
	}
}

func HandleOKX(configFile string, db *sql.DB, stopChan <-chan struct{}, logger *log.Logger) {
	cfg, err := LoadOKXConfig(configFile)
	if err != nil {
		logger.Printf("[OKX] Failed to load config: %v", err)
		return
	}

	go ProcessOKXAccountStream(cfg, db, stopChan, logger)

	<-stopChan
}
