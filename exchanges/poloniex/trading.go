package poloniex

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// TradingProvider represents poloniex trading provider structure
type TradingProvider struct {
	credentials schemas.Credentials
	httpProxy   proxy.Provider
	httpClient  *httpclient.Client
}

// NewTradingProvider - TradingProvider constructor
func NewTradingProvider(credentials schemas.Credentials, httpProxy proxy.Provider) *TradingProvider {
	proxyClient := httpProxy.NewClient(exchangeName)
	return &TradingProvider{
		credentials: credentials,
		httpProxy:   httpProxy,
		httpClient:  httpclient.NewSigned(credentials, proxyClient),
	}
}

// Subscribe subscribing to user trade data updates: balance, orders, trades
func (trading *TradingProvider) Subscribe(interval time.Duration) (chan schemas.UserInfoChannel, chan schemas.UserOrdersChannel, chan schemas.UserTradesChannel) {
	uic := make(chan schemas.UserInfoChannel)
	uoc := make(chan schemas.UserOrdersChannel)
	utc := make(chan schemas.UserTradesChannel)

	if interval < 5*time.Second {
		interval = 5 * time.Second
	}

	go func() {
		for {
			ui, err := trading.Info()
			uic <- schemas.UserInfoChannel{
				Data:  ui,
				Error: err,
			}

			uo, err := trading.Orders([]schemas.Symbol{})
			uoc <- schemas.UserOrdersChannel{
				Data:  uo,
				Error: err,
			}

			ut, _, err := trading.Trades(schemas.FilterOptions{})
			utc <- schemas.UserTradesChannel{
				Data:  ut,
				Error: err,
			}

			time.Sleep(interval)
		}
	}()

	return uic, uoc, utc
}

// Info provides user balance data
func (trading *TradingProvider) Info() (ui schemas.UserInfo, err error) {
	var resp map[string]UserBalance
	var b []byte

	userBalance := make(map[string]schemas.Balance)

	payload := httpclient.Params()
	nonce := time.Now().UnixNano()
	payload.Set("nonce", strconv.FormatInt(nonce, 10))
	payload.Set("command", commandBalance)

	b, err = trading.httpClient.Post(tradingAPI, httpclient.Params(), payload, true)
	if err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	for coin, value := range resp {
		userBalance[coin] = value.Map(coin)
	}

	ui.Balances = userBalance
	return
}

// Orders provides user orders data
func (trading *TradingProvider) Orders(symbols []schemas.Symbol) (orders []schemas.Order, err error) {
	if len(symbols) > 0 {
		for _, symb := range symbols {
			ordrs, err := trading.ordersBySymbol(symb.OriginalName)
			if err != nil {
				return nil, err
			}
			orders = append(orders, ordrs...)
		}
		return
	}

	return trading.allOrders()
}

// Trades provides user trades data
func (trading *TradingProvider) Trades(opts schemas.FilterOptions) (trades []schemas.Trade, p schemas.Paging, err error) {
	if len(opts.Symbols) > 0 {
		for _, symb := range opts.Symbols {
			res, err := trading.tradesBySymbol(symb.OriginalName, opts)
			if err != nil {
				return nil, schemas.Paging{}, err
			}
			trades = append(trades, res...)
		}

		return
	}

	return trading.allTrades(opts)
}

// ImportTrades importing trades by params
func (trading *TradingProvider) ImportTrades(opts schemas.FilterOptions) chan schemas.UserTradesChannel {
	ch := make(chan schemas.UserTradesChannel)
	return ch
}

// Create creating new limit order
func (trading *TradingProvider) Create(order schemas.Order) (result schemas.Order, err error) {
	var b []byte
	var command string
	var resp OrderCreate

	if strings.ToUpper(order.Type) == typeBuy {
		command = commandBuy
	}
	if strings.ToUpper(order.Type) == typeSell {
		command = commandSell
	}

	symbol := unparseSymbol(order.Symbol)
	nonce := time.Now().UnixNano()

	payload := httpclient.Params()
	payload.Set("command", command)
	payload.Set("nonce", strconv.FormatInt(nonce, 10))
	payload.Set("currencyPair", symbol)
	payload.Set("rate", strconv.FormatFloat(order.Price, 'f', -1, 64))
	payload.Set("amount", strconv.FormatFloat(order.Amount, 'f', -1, 64))

	b, err = trading.httpClient.Post(tradingAPI, httpclient.Params(), payload, true)
	if err != nil {
		err = fmt.Errorf("[POLONIEX] Error creating order: %v", string(b))
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if len(resp.Error) > 0 {
		err = fmt.Errorf("[POLONIEX] Error creating order: %v", resp.Error)
		return
	}

	result = order
	result.ID = resp.OrderNumber
	return
}

// Cancel cancelling open order
func (trading *TradingProvider) Cancel(order schemas.Order) (err error) {
	var b []byte
	var resp OrderCancel

	payload := httpclient.Params()
	nonce := time.Now().UnixNano()
	payload.Set("orderNumber", order.ID)
	payload.Set("command", commandCancel)
	payload.Set("nonce", strconv.FormatInt(nonce, 10))

	b, err = trading.httpClient.Post(tradingAPI, httpclient.Params(), payload, true)
	if err != nil {
		err = fmt.Errorf("[POLONIEX] Error creating order: %v", string(b))
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if len(resp.Error) > 0 {
		err = fmt.Errorf("[POLONIEX] Error cancelling order: %v", resp.Error)
		return
	}

	return nil
}

// CancelAll cancelling all open orders
func (trading *TradingProvider) CancelAll() (err error) {
	var orders []schemas.Order

	if orders, err = trading.allOrders(); err != nil {
		return
	}
	for _, ord := range orders {
		err = trading.Cancel(ord)
		if err != nil {
			return err
		}
	}

	return
}

func (trading *TradingProvider) allOrders() (orders []schemas.Order, err error) {
	var resp map[string][]UserOrder
	var b []byte

	payload := httpclient.Params()
	nonce := time.Now().UnixNano()
	payload.Set("nonce", strconv.FormatInt(nonce, 10))
	payload.Set("command", commandPrivateOrders)
	payload.Set("currencyPair", "all")

	b, err = trading.httpClient.Post(tradingAPI, httpclient.Params(), payload, true)
	if err != nil {
		return
	}
	if len(b) == 2 {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	for symb, ords := range resp {
		for _, ord := range ords {
			s, _, _ := parseSymbol(symb)
			orders = append(orders, ord.Map(s))
		}
	}

	return
}

func (trading *TradingProvider) ordersBySymbol(symbol string) (orders []schemas.Order, err error) {
	var resp []UserOrder
	var b []byte

	payload := httpclient.Params()
	nonce := time.Now().UnixNano()
	payload.Set("nonce", strconv.FormatInt(nonce, 10))
	payload.Set("command", commandPrivateOrders)
	payload.Set("currencyPair", symbol)

	b, err = trading.httpClient.Post(tradingAPI, httpclient.Params(), payload, true)
	if err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	for _, ord := range resp {
		s, _, _ := parseSymbol(symbol)
		orders = append(orders, ord.Map(s))
	}

	return
}

func (trading *TradingProvider) tradesBySymbol(symbol string, opts schemas.FilterOptions) (trades []schemas.Trade, err error) {
	var resp []UserTrade
	var b []byte

	payload := httpclient.Params()
	nonce := time.Now().UnixNano()
	payload.Set("nonce", strconv.FormatInt(nonce, 10))
	payload.Set("command", commandPrivateTrades)
	payload.Set("currencyPair", symbol)

	if opts.Limit > 0 {
		payload.Set("limit", fmt.Sprintf("%d", opts.Limit))
	}
	if opts.Since != 0 {
		payload.Set("start", fmt.Sprintf("%d", opts.Since))
	}
	if opts.Before != 0 {
		payload.Set("end", fmt.Sprintf("%d", opts.Before))
	}

	b, err = trading.httpClient.Post(tradingAPI, httpclient.Params(), payload, true)
	if err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	for _, trd := range resp {
		s, _, _ := parseSymbol(symbol)
		trades = append(trades, trd.Map(s))
	}

	return
}

func (trading *TradingProvider) allTrades(opts schemas.FilterOptions) (trades []schemas.Trade, paging schemas.Paging, err error) {
	var resp map[string][]UserTrade
	var b []byte

	payload := httpclient.Params()
	nonce := time.Now().UnixNano()
	payload.Set("nonce", strconv.FormatInt(nonce, 10))
	payload.Set("command", commandPrivateTrades)
	payload.Set("currencyPair", "all")

	if opts.Limit > 0 {
		payload.Set("limit", fmt.Sprintf("%d", opts.Limit))
	}
	if opts.Since != 0 {
		payload.Set("start", fmt.Sprintf("%d", opts.Since))
	}
	if opts.Before != 0 {
		payload.Set("end", fmt.Sprintf("%d", opts.Before))
	}

	b, err = trading.httpClient.Post(tradingAPI, httpclient.Params(), payload, true)
	if err != nil {
		return
	}
	if len(b) == 2 {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	for symb, trds := range resp {
		for _, trd := range trds {
			s, _, _ := parseSymbol(symb)
			trades = append(trades, trd.Map(s))
		}
	}

	return
}
