package binance

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

const (
	httpURL           = "https://api.binance.com/api/v1/userDataStream"
	userDataStreamURL = "wss://stream.binance.com:9443/ws/"

	balanceType   = "outboundAccountInfo"
	executionType = "executionReport"
)

// TradingProvider - provides quotes/ticker
type TradingProvider struct {
	credentials schemas.Credentials
	symbols     []schemas.Symbol
	listenKey   string
	httpProxy   proxy.Provider
	httpClient  *httpclient.Client
	wsClient    *websocket.Client
	uic         chan schemas.UserInfoChannel
	uoc         chan schemas.UserOrdersChannel
	utc         chan schemas.UserTradesChannel
	ch          chan []byte
	ech         chan error
}

// NewTradingProvider - TradingProvider constructor
func NewTradingProvider(credentials schemas.Credentials, httpProxy proxy.Provider) *TradingProvider {
	proxyClient := httpProxy.NewClient(exchangeName)
	trading := TradingProvider{
		credentials: credentials,
		httpProxy:   httpProxy,
		httpClient:  httpclient.NewSigned(credentials, proxyClient),
		uic:         make(chan schemas.UserInfoChannel),
		uoc:         make(chan schemas.UserOrdersChannel),
		utc:         make(chan schemas.UserTradesChannel),
		ch:          make(chan []byte, 400),
		ech:         make(chan error, 400),
	}
	lk, err := trading.CreateListenkey(credentials.APIKey)
	if err != nil {
		log.Println("[BINANCE] Error creating key", err)
	}
	trading.listenKey = lk

	go func() {
		for {
			trading.Ping()
			time.Sleep(30 * time.Minute)
		}
	}()
	trading.wsClient = websocket.NewClient(userDataStreamURL+trading.listenKey, httpProxy)
	// ws updates of trading data

	return &trading
}

// SetSymbols update symbols in trading provider
func (trading *TradingProvider) SetSymbols(symbols []schemas.Symbol) schemas.TradingProvider {
	trading.symbols = symbols

	return trading
}

type errorMsg struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// Price for symbol
type price struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

/*
Subscribe - subscribing to user info
— user info
- orders
- trades
*/
func (trading *TradingProvider) Subscribe(interval time.Duration) (chan schemas.UserInfoChannel, chan schemas.UserOrdersChannel, chan schemas.UserTradesChannel) {
	// http snapshots of trading data
	go func() {
		ui, err := trading.Info()
		if err != nil {
			log.Println("[BINANCE] Balances snapshot error:", err)
		}
		trading.uic <- schemas.UserInfoChannel{
			Data:     ui,
			DataType: "s",
			Error:    err,
		}
	}()

	go func() {
		o, err := trading.Orders(trading.symbols)
		if err != nil {
			log.Println("[BINANCE] Orders snapshot error:", err)
		}
		trading.uoc <- schemas.UserOrdersChannel{
			Data:     o,
			DataType: "s",
			Error:    err,
		}
	}()

	go func() {
		t, _, err := trading.Trades(schemas.FilterOptions{Symbols: trading.symbols})
		if err != nil {
			log.Println("[BINANCE] Trades snapshot error:", err)
		}
		trading.utc <- schemas.UserTradesChannel{
			Data:     t,
			DataType: "s",
			Error:    err,
		}
	}()

	trading.wsClient.Connect()
	trading.wsClient.ChangeKeepAlive(false)
	trading.wsClient.Listen(trading.ch, trading.ech)

	// handling ws input data
	go func() {
		for {
			select {
			case data := <-trading.ch:
				trading.handleUpdates(data)
			case err := <-trading.ech:
				log.Println("[BINANCE] Error handling", err)
				trading.uic <- schemas.UserInfoChannel{
					Data:     schemas.UserInfo{},
					DataType: "u",
					Error:    err,
				}
			}
		}
	}()

	return trading.uic, trading.uoc, trading.utc
}

// Info - provides user info: Keys access, balances
func (trading *TradingProvider) Info() (ui schemas.UserInfo, err error) {
	var b []byte
	var eMsg errorMsg
	params := httpclient.Params()
	params.Set("timestamp", strconv.FormatInt(time.Now().UTC().UnixNano(), 10)[:13])

	b, err = trading.httpClient.Get(apiUserBalance, params, true)
	if err != nil {
		if e := json.Unmarshal(b, &eMsg); e != nil {
			return
		}
		err = errors.New(eMsg.Message)
		return
	}
	var resp UserBalanceResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}

	prices, err := trading.prices()
	if err != nil {
		log.Println("Error getting prices for balances")
	}

	return resp.Map(prices), nil
}

func (trading *TradingProvider) prices() (resp map[string]float64, err error) {
	var b []byte
	var eMsg errorMsg

	b, err = trading.httpClient.Get(apiPrices, httpclient.Params(), false)
	if err != nil {
		if e := json.Unmarshal(b, &eMsg); e != nil {
			return
		}
		err = errors.New(eMsg.Message)
		return
	}

	var prices []price
	if err = json.Unmarshal(b, &prices); err != nil {
		return
	}

	resp = make(map[string]float64)
	for _, p := range prices {
		symbol, _, _ := parseSymbol(p.Symbol)
		price, err := strconv.ParseFloat(p.Price, 64)
		if err != nil {
			log.Println("Error parsing price", err)
		}
		resp[symbol] = price
	}

	return
}

// Orders - getting user active orders
func (trading *TradingProvider) Orders(symbols []schemas.Symbol) (orders []schemas.Order, err error) {
	var b []byte
	var resp UserOrdersResponse
	var eMsg errorMsg
	params := httpclient.Params()
	params.Set("timestamp", strconv.FormatInt(time.Now().UTC().UnixNano(), 10)[:13])

	b, err = trading.httpClient.Get(apiActiveOrders, params, true)
	if err != nil {
		if e := json.Unmarshal(b, &eMsg); e != nil {
			return
		}
		err = errors.New(eMsg.Message)
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}

	return resp.Map(), nil
}

// Trades - getting user trades
func (trading *TradingProvider) Trades(opts schemas.FilterOptions) (trades []schemas.Trade, p schemas.Paging, err error) {
	var resp UserTradesResponse
	var b []byte
	var result []schemas.Trade
	var eMsg errorMsg

	for _, s := range opts.Symbols {
		params := httpclient.Params()
		params.Set("timestamp", strconv.FormatInt(time.Now().UTC().UnixNano(), 10)[:13])
		params.Set("symbol", s.OriginalName)

		b, err = trading.httpClient.Get(apiUserTrades, params, true)
		if err != nil {
			if e := json.Unmarshal(b, &eMsg); e != nil {
				return
			}
			err = errors.New(eMsg.Message)
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			return
		}
		respSymb := resp.Map()
		result = append(result, respSymb...)
	}
	return result, schemas.Paging{}, nil
}

// handleUpdates - handling incoming updates data
func (trading *TradingProvider) handleUpdates(data []byte) {
	log.Println("[BINANCE] INCOMING WS DATA:", string(data))
	var msg generalMessage
	err := json.Unmarshal(data, &msg)
	if err != nil {
		log.Println("[BINANCE] Unmarshalling error:", err)
	}

	if msg.EventType == balanceType {
		var balanceMsg balanceMessage
		err = json.Unmarshal(data, &balanceMsg)
		if err != nil {
			log.Println("[BINANCE] Balance unmarshalling error:", err)
		}
		ui := balanceMsg.Map()
		trading.uic <- schemas.UserInfoChannel{
			Data:  ui,
			Error: err,
		}
	}

	if msg.EventType == executionType {
		var tradesMsg tradesMessage
		err = json.Unmarshal(data, &tradesMsg)
		if err != nil {
			log.Println("[BINANCE] Trades unmarshalling error:", err)
		}

		if tradesMsg.CurrentExecutionType == "TRADE" {
			t := tradesMsg.Map()
			trading.utc <- schemas.UserTradesChannel{
				Data:  t,
				Error: err,
			}
		}

		o := tradesMsg.MapOrder()
		trading.uoc <- schemas.UserOrdersChannel{
			Data:  o,
			Error: err,
		}
	}
}

// ImportTrades importing trades
func (trading *TradingProvider) ImportTrades(opts schemas.FilterOptions) chan schemas.UserTradesChannel {
	ch := make(chan schemas.UserTradesChannel)
	if len(fmt.Sprintf("%d", opts.Before)) < 12 {
		opts.Before = opts.Before * 1000
	}
	if len(fmt.Sprintf("%d", opts.Since)) < 12 {
		opts.Since = opts.Since * 1000
	}

	trades, paging, err := trading.Trades(opts)
	opts.Page = int(paging.Pages)
	go func() {
		for {
			trades, _, err := trading.Trades(opts)
			if err != nil {
				log.Println("[BINANCE] Error loading trades: ", err)
				continue
			}
			ch <- schemas.UserTradesChannel{
				Data:  trades,
				Error: err,
			}
			opts.Page = opts.Page - 1
			if opts.Page < 1 {
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()
	log.Printf("paging: %d, %+v, %v", len(trades), paging, err)

	return ch
}

// Create - creating order
func (trading *TradingProvider) Create(order schemas.Order) (result schemas.Order, err error) {
	var b []byte
	var eMsg errorMsg
	query := httpclient.Params()

	query.Set("symbol", unparseSymbol(order.Symbol))
	query.Set("type", "LIMIT")
	query.Set("timeInForce", "GTC")
	query.Set("side", strings.ToUpper(order.Type))
	query.Set("price", strconv.FormatFloat(order.Price, 'f', -1, 64))
	query.Set("quantity", strconv.FormatFloat(order.Amount, 'f', -1, 64))
	query.Set("timestamp", strconv.FormatInt(time.Now().UnixNano(), 10)[:13])

	b, err = trading.httpClient.Post(apiCreateOrder, query, httpclient.KeyValue{}, true)
	if err != nil {
		if e := json.Unmarshal(b, &eMsg); e != nil {
			return
		}
		err = errors.New(eMsg.Message)
		return
	}
	var resp OrderCreateResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	price, err := strconv.ParseFloat(resp.Price, 64)
	if err != nil {
		log.Println("[BINANCE] Error mapping price in private trades. Binance:", err)
	}
	amount, err := strconv.ParseFloat(resp.OriginalQuantity, 64)
	if err != nil {
		log.Println("[BINANCE] Error mapping qty in private trades. Binance:", err)
	}
	amountFilled, err := strconv.ParseFloat(resp.ExecQuantity, 64)
	if err != nil {
		log.Println("[BINANCE] Error mapping filled qty in private trades. Binance:", err)
	}
	result = schemas.Order{
		ID:           strconv.FormatInt(resp.OrderID, 10),
		Symbol:       resp.Symbol,
		Type:         resp.Side,
		Price:        price,
		Amount:       amount,
		AmountFilled: amountFilled,
		Count:        1,
		CreatedAt:    resp.Time,
		Remove:       0,
	}
	return
}

// Cancel - cancelling order
func (trading *TradingProvider) Cancel(order schemas.Order) (err error) {
	var b []byte
	var eMsg errorMsg

	query := httpclient.Params()
	query.Set("symbol", unparseSymbol(order.Symbol))
	query.Set("orderId", order.ID)
	query.Set("timestamp", strconv.FormatInt(time.Now().UnixNano(), 10)[:13])

	b, err = trading.httpClient.Request("DELETE", apiCancelOrder, query, httpclient.Params(), true)
	if err != nil {
		if e := json.Unmarshal(b, &eMsg); e != nil {
			return
		}
		err = errors.New(eMsg.Message)
		return
	}
	var resp OrderCancelResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}

	return
}

// CancelAll - cancelling all orders
func (trading *TradingProvider) CancelAll() (err error) {
	var orders []schemas.Order
	if orders, err = trading.Orders([]schemas.Symbol{}); err != nil {
		return
	}
	for _, o := range orders {
		err = trading.Cancel(o)
	}
	return
}
