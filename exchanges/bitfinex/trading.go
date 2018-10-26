package bitfinex

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

const (
	errConnecting    = "[BITFINEX] Error connecting to bitfinex WS: %v"
	errAuth          = "[BITFINEX] Bitfinex auth error: %v"
	errOnWs          = "[BITFINEX] Error from websocket client: %v"
	errLoadingTrades = "[BITFINEX] Error loading trades: %v"
	errWsNotAuth     = "[BITFINEX] WS subscription not authorized"
	errUnmarshal     = "[BITFINEX] Error unmarshalling message: %v"
	errExitWsClient  = "[BITFINEX] Error exiting WS client: %v"
	errCancelAll     = "[BITFINEX] Error cancelling all orders: %v"
	errCreateOrder   = "[BITFINEX] Error creating order: %v"
	errCancelOrder   = "[BITFINEX] Error cancelling order: %v"
)

const (
	codeRestart   = 20051
	codeMaintance = 20060
)

const (
	cancelAllStatus = "All orders cancelled"
)

// TradingProvider represents bitfinex trading provider structure
type TradingProvider struct {
	credentials schemas.Credentials
	wsClient    *websocket.Client
	httpClient  *httpclient.Client
	proxyClient proxy.Client

	bus     tradingBus
	symbols []schemas.Symbol
}

type tradingBus struct {
	uic chan schemas.UserInfoChannel
	uoc chan schemas.UserOrdersChannel
	utc chan schemas.UserTradesChannel
}

// NewTradingProvider constructing bitfinex trading provider
func NewTradingProvider(creds schemas.Credentials, proxy proxy.Provider) *TradingProvider {
	proxyClient := proxy.NewClient(exchangeName)
	wsClient := websocket.NewClient(wsURL, proxy)

	return &TradingProvider{
		credentials: creds,
		wsClient:    wsClient,
		httpClient:  httpclient.NewSigned(creds, proxyClient),
		proxyClient: proxyClient,
		bus: tradingBus{
			uic: make(chan schemas.UserInfoChannel, 100),
			uoc: make(chan schemas.UserOrdersChannel, 100),
			utc: make(chan schemas.UserTradesChannel, 100),
		},
	}
}

// SetSymbols update symbols in trading provider
func (trading *TradingProvider) SetSymbols(symbols []schemas.Symbol) schemas.TradingProvider {
	trading.symbols = symbols

	return trading
}

// Subscribe subscribing to accounts updates for balances, orders, trades
func (trading *TradingProvider) Subscribe(interval time.Duration) (chan schemas.UserInfoChannel, chan schemas.UserOrdersChannel, chan schemas.UserTradesChannel) {
	trading.subscribe()

	return trading.bus.uic, trading.bus.uoc, trading.bus.utc
}

// Unsubscribe from trading data
func (trading *TradingProvider) Unsubscribe() error {
	return trading.wsClient.Exit()
}

// Info stub method
func (trading *TradingProvider) Info() (ui schemas.UserInfo, err error) {
	var b []byte
	var resp []interface{}

	payload := make(map[string]interface{})
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	path := "/v2/auth/r/wallets"
	req, err := http.NewRequest("POST", apiURL+path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return
	}
	signedReq := signV2(trading.credentials.APIKey, trading.credentials.APISecret, path, req)
	b, err = trading.httpClient.Do(signedReq)
	if err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}

	balances := trading.mapBalance(resp)

	access, err := trading.getAccessInfo()
	if err != nil {
		return
	}

	prices, err := trading.prices()
	if err != nil {
		log.Println("Error getting prices for symbols", err)
	}

	ui = schemas.UserInfo{
		Access:   access,
		Balances: balances,
		Prices:   prices,
	}

	return
}

func (trading *TradingProvider) prices() (resp map[string]float64, err error) {
	var b []byte

	path := "/v2/tickers"
	params := httpclient.Params()
	params.Set("symbols", "ALL")
	b, err = trading.httpClient.Get(apiURL+path, params, false)
	if err != nil {
		return
	}

	var prices [][]interface{}
	if err = json.Unmarshal(b, &prices); err != nil {
		return
	}

	// log.Println(string(b))
	// log.Println("=====================")
	// log.Println(prices)

	resp = make(map[string]float64)
	for _, p := range prices {
		var symbol string
		if symb, ok := p[0].(string); ok {
			if strings.Index(symb, "f") != 0 { // for symbols as fUSD, fSAN, etc.
				symbol, _, _ = parseSymbol(symb)
			}
		}
		if price, ok := p[7].(float64); ok {
			resp[symbol] = price
		}
	}

	return
}

// Orders stub method
func (trading *TradingProvider) Orders(symbols []schemas.Symbol) (orders []schemas.Order, err error) {
	var b []byte
	var resp []interface{}

	for _, symb := range symbols {
		payload := make(map[string]interface{})
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		path := "/v2/auth/r/orders/" + symb.OriginalName
		req, err := http.NewRequest("POST", apiURL+path, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return nil, err
		}
		signedReq := signV2(trading.credentials.APIKey, trading.credentials.APISecret, path, req)
		b, err = trading.httpClient.Do(signedReq)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			return nil, err
		}

		ordr := trading.mapOrders(resp)
		if len(ordr) > 0 {
			orders = append(orders, ordr[0])
		}
		time.Sleep(200 * time.Millisecond)
	}

	return
}

// Trades stub method
func (trading *TradingProvider) Trades(opts schemas.FilterOptions) (trades []schemas.Trade, p schemas.Paging, err error) {
	var b []byte
	var resp []interface{}

	payload := make(map[string]interface{})
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	path := "/v2/auth/r/trades/hist"
	req, err := http.NewRequest("POST", apiURL+path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return
	}

	query := req.URL.Query()
	if opts.Limit > 0 {
		query.Add("limit", fmt.Sprintf("%d", opts.Limit))
	}
	if opts.Since != 0 {
		query.Add("start", fmt.Sprintf("%d", opts.Since))
	}
	if opts.Before != 0 {
		query.Add("end", fmt.Sprintf("%d", opts.Before))
	}
	req.URL.RawQuery = query.Encode()

	signedReq := signV2(trading.credentials.APIKey, trading.credentials.APISecret, path, req)
	b, err = trading.httpClient.Do(signedReq)
	if err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}

	trades = trading.mapTrades(resp)
	return
}

// ImportTrades stub method
func (trading *TradingProvider) ImportTrades(opts schemas.FilterOptions) chan schemas.UserTradesChannel {
	ch := make(chan schemas.UserTradesChannel)
	return ch
}

// Create stub method
func (trading *TradingProvider) Create(order schemas.Order) (result schemas.Order, err error) {
	var b []byte
	var orderType string
	var resp newOrderResponse

	symbol := unparseSymbol(order.Symbol)
	if strings.ToUpper(order.Type) == schemas.TypeBuy {
		orderType = "buy"
	}
	if strings.ToUpper(order.Type) == schemas.TypeSell {
		orderType = "sell"
	}

	// nonce := strconv.FormatInt(time.Now().UnixNano(), 10)[:13]
	nonce := fmt.Sprintf("%v", time.Now().UnixNano()/1000)

	payload := map[string]interface{}{
		"request": "/v1/order/new",
		"nonce":   nonce,
		"symbol":  symbol,
		"amount":  strconv.FormatFloat(order.Amount, 'f', -1, 64),
		"price":   strconv.FormatFloat(order.Price, 'f', -1, 64),
		"side":    orderType,
		"type":    "exchange limit", // TODO: add type to order model, handle it here
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}
	req, err := http.NewRequest("POST", apiNewOrder, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return
	}
	signedReq := signV1(trading.credentials.APIKey, trading.credentials.APISecret, req)
	b, err = trading.httpClient.Do(signedReq)
	if err != nil {
		err = errors.New(string(b))
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}

	var side string
	var price, amount float64
	smb, _, _ := parseSymbol("t" + strings.ToUpper(resp.Symbol))

	if resp.Side == "buy" {
		side = schemas.TypeBuy
	}
	if resp.Side == "sell" {
		side = schemas.TypeSell
	}

	status := schemas.StatusNew
	if resp.IsCancelled {
		status = schemas.StatusCancelled
	}

	price, _ = strconv.ParseFloat(resp.Price, 64)
	amount, _ = strconv.ParseFloat(resp.OriginalAmount, 64)
	tms, _ := strconv.ParseInt(resp.Timestamp, 10, 64)

	result = schemas.Order{
		ID:        strconv.FormatInt(resp.ID, 10),
		Symbol:    smb,
		Type:      side,
		Price:     price,
		Amount:    amount,
		CreatedAt: tms,
		Status:    status,
	}

	return
}

// Cancel stub method
func (trading *TradingProvider) Cancel(order schemas.Order) (err error) {
	var b []byte
	var resp newOrderResponse

	// nonce := strconv.FormatInt(time.Now().UnixNano(), 10)[:13]
	nonce := fmt.Sprintf("%v", time.Now().UnixNano()/1000)

	orderID, _ := strconv.ParseInt(order.ID, 10, 64)
	payload := map[string]interface{}{
		"request":  "/v1/order/cancel",
		"nonce":    nonce,
		"order_id": orderID,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}
	req, err := http.NewRequest("POST", apiCancelOrder, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return
	}
	signedReq := signV1(trading.credentials.APIKey, trading.credentials.APISecret, req)
	b, err = trading.httpClient.Do(signedReq)
	if err != nil {
		err = errors.New(string(b))
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}

	return
}

// CancelAll stub method
func (trading *TradingProvider) CancelAll() (err error) {
	var b []byte
	var resp cancelAllResponse

	nonce := fmt.Sprintf("%v", time.Now().UnixNano()/1000)

	payload := map[string]interface{}{
		"request": "/v1/order/cancel/all",
		"nonce":   nonce,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}
	req, err := http.NewRequest("POST", apiCancelOrder, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return
	}
	signedReq := signV1(trading.credentials.APIKey, trading.credentials.APISecret, req)
	b, err = trading.httpClient.Do(signedReq)
	if err != nil {
		return
	}
	b, err = trading.httpClient.Do(req)
	if err != nil {
		err = errors.New(string(b))
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if resp.Result != cancelAllStatus {
		err = fmt.Errorf(errCancelAll, resp.Result)
		return
	}

	return
}

func (trading *TradingProvider) subscribe() {
	dch := make(chan []byte, 100)
	ech := make(chan error, 100)

	if err := trading.wsClient.Connect(); err != nil {
		err = fmt.Errorf(errConnecting, err)
		trading.publishErr(err)

		// resubscribing on connection error
		trading.resubscribe()
		return
	}
	trading.wsClient.ChangeKeepAlive(false)
	trading.wsClient.Listen(dch, ech)

	go func() {
		// bitfinex hasn't got trades snapshot on websockets
		// so we need to get snapshot by HTTP.
		// We need sleep so that nonce on HTTP and
		// ws auth wiil be different
		time.Sleep(1 * time.Second)
		trades, _, err := trading.Trades(schemas.FilterOptions{})
		if err != nil {
			log.Printf(errLoadingTrades, err)
			err = fmt.Errorf(errLoadingTrades, err)
			trading.publishErr(err)
		}
		trading.bus.utc <- schemas.UserTradesChannel{
			Data: trades,
		}
	}()
	go func() {
		for {
			select {
			case msg := <-dch:
				log.Println("Incoming message: ", string(msg))
				trading.handleMessages(msg)
			case err := <-ech:
				log.Printf(errOnWs, err)
				err = fmt.Errorf(errOnWs, err)
				trading.publishErr(err)
			}
		}
	}()

	if err := trading.auth(); err != nil {
		log.Printf(errAuth, err)
		err = fmt.Errorf(errAuth, err)
		trading.publishErr(err)

		// resubscribing on auth error
		trading.resubscribe()
		return
	}
}

func (trading *TradingProvider) resubscribe() {
	time.Sleep(1 * time.Second)
	if err := trading.wsClient.Exit(); err != nil {
		log.Printf(errExitWsClient, err)
	}

	trading.subscribe()
	return
}

func (trading *TradingProvider) auth() error {
	nonce := fmt.Sprintf("%v", time.Now().UnixNano()/1000)

	payload := "AUTH" + nonce
	signature := createSignature384(payload, trading.credentials.APISecret)

	msg := authMsg{
		Event:       "auth",
		APIKey:      trading.credentials.APIKey,
		AuthSig:     signature,
		AuthPayload: payload,
		AuthNonce:   nonce,
	}

	return trading.wsClient.Write(msg)
}

func (trading *TradingProvider) handleMessages(data []byte) {
	var msg interface{}

	err := json.Unmarshal(data, &msg)
	if err != nil {
		log.Printf(errUnmarshal, err)
		return
	}

	if eventMsg, ok := msg.(map[string]interface{}); ok {
		if eventMsg["event"] != nil {
			if eventMsg["event"] == "auth" {
				err = trading.checkAuthMessage(eventMsg)
				if err != nil {
					trading.publishErr(err)
					return
				}
			} else {
				err := trading.handleEvents(eventMsg)
				if err != nil {
					trading.publishErr(err)
					return
				}
			}
		}
	}

	if updateMsg, ok := msg.([]interface{}); ok {
		trading.handleUpdates(updateMsg)
	}
}

func (trading *TradingProvider) handleUpdates(msg []interface{}) {
	updType := msg[1]

	if updType == "ws" {
		b := trading.mapBalance(msg[2].([]interface{}))
		access, err := trading.getAccessInfo()
		if err != nil {
			trading.publishErr(err)
			return
		}

		prices, err := trading.prices()
		if err != nil {
			log.Println("Error getting prices for symbols", err)
		}

		trading.bus.uic <- schemas.UserInfoChannel{
			DataType: dataTypeSnapshot,
			Data: schemas.UserInfo{
				Access:   access,
				Balances: b,
				Prices:   prices,
			},
		}
	}
	if updType == "wu" {
		wslice := []interface{}{msg[2]}
		b := trading.mapBalance(wslice)
		access, err := trading.getAccessInfo()
		if err != nil {
			trading.publishErr(err)
			return
		}

		trading.bus.uic <- schemas.UserInfoChannel{
			DataType: dataTypeUpdate,
			Data: schemas.UserInfo{
				Access:   access,
				Balances: b,
			},
		}
	}
	if updType == "os" {
		m := trading.mapOrders(msg[2].([]interface{}))
		trading.bus.uoc <- schemas.UserOrdersChannel{
			DataType: dataTypeSnapshot,
			Data:     m,
		}
	}
	if updType == "on" || updType == "ou" || updType == "oc" {
		wslice := []interface{}{msg[2]}
		m := trading.mapOrders(wslice)
		trading.bus.uoc <- schemas.UserOrdersChannel{
			DataType: dataTypeUpdate,
			Data:     m,
		}
	}
	if updType == "tu" {
		m := trading.mapTrades(msg[2].([]interface{}))
		trading.bus.utc <- schemas.UserTradesChannel{
			DataType: dataTypeUpdate,
			Data:     m,
		}
	}
}

func (trading *TradingProvider) handleEvents(msg map[string]interface{}) error {
	if msg["event"] == "error" {
		log.Println("WS error: ", msg)
		return errors.New("WS error: " + msg["msg"].(string))
	}
	if msg["event"] == "info" {
		if msg["code"] == codeRestart {
			trading.resubscribe()

			return nil
		}
		if msg["code"] == codeMaintance {
			time.Sleep(120 * time.Second)
			trading.resubscribe()

			return nil
		}
	}

	return nil
}

func (trading *TradingProvider) checkAuthMessage(msg map[string]interface{}) error {
	if msg["status"] == "OK" {
		log.Println("WS auth is ok")
		return nil
	}

	return errors.New(errWsNotAuth)
}

func (trading *TradingProvider) getAccessInfo() (access schemas.Access, err error) {
	var b []byte
	var resp accessResponse

	nonce := fmt.Sprintf("%v", time.Now().UnixNano()/1000)

	payload := map[string]interface{}{
		"request": "/v1/key_info",
		"nonce":   nonce,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}
	req, err := http.NewRequest("POST", apiAccess, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return
	}
	signedReq := signV1(trading.credentials.APIKey, trading.credentials.APISecret, req)
	b, err = trading.httpClient.Do(signedReq)
	if err != nil {
		return
	}

	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}

	access.Read = resp.Account.Read
	access.Trade = resp.Orders.Write
	access.Withdraw = resp.Withdraw.Write

	return
}

func (trading *TradingProvider) publishErr(err error) {
	go func() {
		trading.bus.uic <- schemas.UserInfoChannel{
			Error: err,
		}
	}()
}

func (trading *TradingProvider) mapBalance(msg []interface{}) map[string]schemas.Balance {
	sb := make(map[string]schemas.Balance)

	for i := range msg {
		if wal, ok := msg[i].([]interface{}); ok {
			b := schemas.Balance{
				Coin:  wal[1].(string),
				Total: wal[2].(float64),
			}
			// according to Bitfinex docs `available` balance field can be null
			// if value isn't fresh enough
			if wal[4] == nil {
				b.Available = b.Total
				b.InOrders = b.Total - b.Available
			}
			if aval, ok := wal[4].(float64); ok {
				b.Available = aval
				b.InOrders = b.Total - b.Available
			}

			sb[wal[1].(string)] = b
		}
	}

	return sb
}

func (trading *TradingProvider) mapOrders(msg []interface{}) (orders []schemas.Order) {
	for i := range msg {
		if ord, ok := msg[i].([]interface{}); ok {
			var side, status string

			symbol, _, _ := parseSymbol(ord[3].(string))

			if ord[6].(float64) > 0 {
				side = schemas.TypeBuy
			} else {
				side = schemas.TypeSell
			}

			if ord[13] == "EXECUTED" {
				status = schemas.StatusTrade
			} else if ord[13] == "ACTIVE" {
				status = schemas.StatusNew
			} else if ord[13] == "CANCELED" {
				status = schemas.StatusCancelled
			} else if ord[13] == "REJECTED" {
				status = schemas.StatusRejected
			} else {
				// if st, ok := ord[13].(string); ok {
				// 	status = st
				// }
				status = schemas.StatusNew
			}

			order := schemas.Order{
				ID:        strconv.FormatFloat(ord[0].(float64), 'f', -1, 64),
				Symbol:    symbol,
				Type:      side,
				Status:    status,
				Price:     ord[16].(float64),
				Amount:    math.Abs(ord[6].(float64)),
				CreatedAt: int64(ord[4].(float64)),
			}

			orders = append(orders, order)
		}
	}

	return
}

func (trading *TradingProvider) mapTrades(msg []interface{}) (trades []schemas.Trade) {
	for i := range msg {
		if trd, ok := msg[i].([]interface{}); ok {
			var side string
			var fee float64

			symbol, _, _ := parseSymbol(trd[1].(string))

			if trd[4].(float64) > 0 {
				side = schemas.TypeBuy
			} else {
				side = schemas.TypeSell
			}

			if f, ok := trd[9].(float64); ok {
				fee = f
			}

			trade := schemas.Trade{
				ID:        strconv.FormatFloat(trd[0].(float64), 'f', -1, 64),
				OrderID:   strconv.FormatFloat(trd[3].(float64), 'f', -1, 64),
				Symbol:    symbol,
				Type:      side,
				Timestamp: int64(trd[2].(float64)),
				Amount:    math.Abs(trd[4].(float64)),
				Price:     trd[5].(float64),
				Fee:       fee,
			}

			trades = append(trades, trade)
		}
	}

	return
}
