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
	httpURL = "https://api.binance.com/api/v1/userDataStream"

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
}

// NewTradingProvider - TradingProvider constructor
func NewTradingProvider(credentials schemas.Credentials, httpProxy proxy.Provider, symbols []schemas.Symbol) *TradingProvider {
	proxyClient := httpProxy.NewClient(exchangeName)
	return &TradingProvider{
		credentials: credentials,
		httpProxy:   httpProxy,
		httpClient:  httpclient.NewSigned(credentials, proxyClient),
		wsClient:    websocket.NewClient(wsURL, httpProxy),
		uic:         make(chan schemas.UserInfoChannel),
		uoc:         make(chan schemas.UserOrdersChannel),
		utc:         make(chan schemas.UserTradesChannel),
	}
}

// Info - provides user info: Keys access, balances
func (trading *TradingProvider) Info() (ui schemas.UserInfo, err error) {
	var b []byte
	params := httpclient.Params()
	params.Set("timestamp", strconv.FormatInt(time.Now().UTC().UnixNano(), 10)[:13])

	b, err = trading.httpClient.Get(apiUserBalance, params, true)
	if err != nil {
		return
	}
	var resp UserBalanceResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	return resp.Map(), nil
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
		trading.uic <- schemas.UserInfoChannel{
			Data:  ui,
			Error: err,
		}
	}()

	go func() {
		o, err := trading.Orders(trading.symbols)
		trading.uoc <- schemas.UserOrdersChannel{
			Data:  o,
			Error: err,
		}
	}()

	go func() {
		t, _, err := trading.Trades(schemas.FilterOptions{Symbols: trading.symbols})
		trading.utc <- schemas.UserTradesChannel{
			Data:  t,
			Error: err,
		}
	}()

	// ws updates of trading data
	ch := make(chan []byte, 100)
	ech := make(chan error, 100)
	lk, err := trading.CreateListenkey(trading.credentials.APIKey)
	if err != nil {
		log.Println("Error creating key", err)
	}
	trading.listenKey = lk

	go func() {
		for {
			trading.Ping()
			time.Sleep(30 * time.Minute)
		}
	}()

	trading.wsClient.Connect()
	trading.wsClient.ChangeKeepAlive(false)
	trading.wsClient.Listen(ch, ech)
	// handling ws input data
	go func() {
		select {
		case data := <-ch:
			trading.handleUpdates(data)
		case err := <-ech:
			log.Println("Error handling", err)
			trading.uic <- schemas.UserInfoChannel{
				Data:  schemas.UserInfo{},
				Error: err,
			}
		}
	}()

	return trading.uic, trading.uoc, trading.utc
}

// Orders - getting user active orders
func (trading *TradingProvider) Orders(symbols []schemas.Symbol) (orders []schemas.Order, err error) {
	var b []byte
	var resp UserOrdersResponse
	var result []schemas.Order
	params := httpclient.Params()
	params.Set("timestamp", strconv.FormatInt(time.Now().UTC().UnixNano(), 10)[:13])
	for _, s := range symbols {
		params.Set("symbol", s.OriginalName)

		b, err = trading.httpClient.Get(apiActiveOrders, params, true)
		if err != nil {
			return
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			return
		}
		respSymb := resp.Map()
		result = append(result, respSymb...)
	}

	return result, nil
}

// handleUpdates - handling incoming updates data
func (trading *TradingProvider) handleUpdates(data []byte) {
	var msg generalMessage
	err := json.Unmarshal(data, &msg)
	if err != nil {
		log.Println("Unmarshalling error:", err)
	}

	if msg.EventType == balanceType {
		var balanceMsg balanceMessage
		err = json.Unmarshal(data, &balanceMsg)
		if err != nil {
			log.Println("Balance unmarshalling error:", err)
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
			log.Println("Trades unmarshalling error:", err)
		}

		if tradesMsg.CurrentExecutionType == "TRADE" {
			if tradesMsg.CurrentOrderStatus == "FILLED" {
				o := tradesMsg.MapOrder()
				trading.uoc <- schemas.UserOrdersChannel{
					Data:  o,
					Error: err,
				}
			}
		}
		t := tradesMsg.Map()
		trading.utc <- schemas.UserTradesChannel{
			Data:  t,
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
				log.Println("Error loading trades: ", err)
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

// Trades - getting user trades
func (trading *TradingProvider) Trades(opts schemas.FilterOptions) (trades []schemas.Trade, p schemas.Paging, err error) {
	var resp UserTradesResponse
	var b []byte
	var result []schemas.Trade

	params := httpclient.Params()
	params.Set("timestamp", strconv.FormatInt(time.Now().UTC().UnixNano(), 10)[:13])
	for _, s := range opts.Symbols {
		params.Set("symbol", s.OriginalName)

		b, err = trading.httpClient.Get(apiUserTrades, params, true)
		if err != nil {
			return
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			return
		}
		respSymb := resp.Map()
		result = append(result, respSymb...)
	}
	return result, schemas.Paging{}, nil
}

// Create - creating order
func (trading *TradingProvider) Create(order schemas.Order) (result schemas.Order, err error) {
	var b []byte
	params := httpclient.Params()

	payload := httpclient.Params()
	payload.Set("symbol", order.Symbol)
	payload.Set("type", strings.ToUpper(order.Type))
	payload.Set("price", fmt.Sprintf("%.10f", order.Price))
	payload.Set("amount", fmt.Sprintf("%.10f", order.Amount))

	b, err = trading.httpClient.Post(apiCreateOrder, params, payload, true)
	if err != nil {
		return
	}
	var resp OrderCreateResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if resp.Success == false {
		err = errors.New(resp.Msg)
		return
	}
	order.ID = resp.Data.OrderOid
	order.CreatedAt = resp.Timestamp
	result = order
	return
}

// Cancel - cancelling order
func (trading *TradingProvider) Cancel(order schemas.Order) (err error) {
	var b []byte

	params := httpclient.Params()
	params.Set("symbol", order.Symbol)

	payload := httpclient.Params()
	payload.Set("symbol", order.Symbol)
	payload.Set("orderOid", order.ID)
	payload.Set("type", order.Type)

	b, err = trading.httpClient.Post(apiCancelOrder, params, payload, true)
	if err != nil {
		return
	}
	var resp OrderCancelResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		log.Println("err: ", err)
		return
	}
	if resp.Success == false {
		return errors.New(resp.Msg)
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
