package tidex

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/syndicatedb/goex/clients"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// TradingProvider - provides quotes/ticker
type TradingProvider struct {
	credentials schemas.Credentials
	httpProxy   proxy.Provider
	httpClient  *clients.HTTP
}

// NewTradingProvider - TradingProvider constructor
func NewTradingProvider(credentials schemas.Credentials, httpProxy proxy.Provider) *TradingProvider {
	proxyClient := httpProxy.NewClient(exchangeName)
	return &TradingProvider{
		credentials: credentials,
		httpProxy:   httpProxy,
		httpClient:  clients.NewSignedHTTP(credentials, proxyClient),
	}
}

// Info - provides user info: Keys access, balances
func (trading *TradingProvider) Info() (ui schemas.UserInfo, err error) {
	var b []byte
	payload := clients.Params()
	payload.Set("method", "getInfoExt")
	payload.Set("nonce", fmt.Sprintf("%d", time.Now().Unix()))

	b, err = trading.httpClient.Post(apiUserInfo, clients.Params(), payload, true)
	if err != nil {
		return
	}
	var resp UserInfoResponse
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
	uic := make(chan schemas.UserInfoChannel)
	uoc := make(chan schemas.UserOrdersChannel)
	utc := make(chan schemas.UserTradesChannel)

	if interval == 0 {
		interval = SubscriptionInterval
	}
	lastTradeID := "1"
	go func() {
		for {
			ui, err := trading.Info()
			uic <- schemas.UserInfoChannel{
				Data:  ui,
				Error: err,
			}
			o, err := trading.Orders([]schemas.Symbol{})
			uoc <- schemas.UserOrdersChannel{
				Data:  o,
				Error: err,
			}
			t, err := trading.Trades(schemas.TradeHistoryOptions{
				FromID: lastTradeID,
			})
			utc <- schemas.UserTradesChannel{
				Data:  t,
				Error: err,
			}
			time.Sleep(interval)
		}
	}()
	return uic, uoc, utc
}

// Orders - getting user active orders
func (trading *TradingProvider) Orders(symbols []schemas.Symbol) (orders []schemas.Order, err error) {
	var b []byte
	payload := clients.Params()
	payload.Set("method", "ActiveOrders")
	payload.Set("nonce", fmt.Sprintf("%d", time.Now().Unix()))
	if len(symbols) > 0 {
		var pairs []string
		for _, s := range symbols {
			pairs = append(pairs, s.OriginalName)
		}
		payload.Set("pair", strings.Join(pairs, "-"))
	}
	b, err = trading.httpClient.Post(apiUserInfo, clients.Params(), payload, true)
	if err != nil {
		return
	}
	var resp UserOrdersResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	return resp.Map(), nil
}

// Trades - getting user trades
func (trading *TradingProvider) Trades(opts schemas.TradeHistoryOptions) (trades []schemas.Trade, err error) {
	var b []byte
	payload := clients.Params()
	payload.Set("method", "TradeHistory")
	payload.Set("nonce", fmt.Sprintf("%d", time.Now().Unix()))

	if len(opts.Symbols) > 0 {
		var pairs []string
		for _, s := range opts.Symbols {
			pairs = append(pairs, s.OriginalName)
		}
		payload.Set("pair", strings.Join(pairs, "-"))
	}

	if opts.Limit > 0 {
		payload.Set("count", fmt.Sprintf("%d", opts.Limit))
	}

	if opts.FromID != "" {
		payload.Set("from_id", opts.FromID)
	}

	b, err = trading.httpClient.Post(apiUserInfo, clients.Params(), payload, true)
	if err != nil {
		return
	}
	var resp UserTradesResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	return resp.Map(), nil
}

// Create - creating order
func (trading *TradingProvider) Create(order schemas.Order) (result schemas.Order, err error) {
	var b []byte

	payload := clients.Params()
	payload.Set("method", "Trade")
	payload.Set("nonce", fmt.Sprintf("%d", time.Now().Unix()))

	pair := symbolToPair(order.Symbol)
	payload.Set("pair", pair)
	payload.Set("type", strings.ToLower(order.Type))
	payload.Set("rate", fmt.Sprintf("%f", order.Price))
	payload.Set("amount", fmt.Sprintf("%f", order.Amount))

	b, err = trading.httpClient.Post(apiUserInfo, clients.Params(), payload, true)
	if err != nil {
		return
	}
	var resp OrdersCreateResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	order.ID = fmt.Sprintf("%d", resp.Return.OrderID)
	order.CreatedAt = time.Now().UTC().UnixNano() / 1000000
	return order, nil
}

// Cancel - cancelling order
func (trading *TradingProvider) Cancel(order schemas.Order) (err error) {
	var b []byte

	payload := clients.Params()
	payload.Set("method", "CancelOrder")
	payload.Set("nonce", fmt.Sprintf("%d", time.Now().Unix()))

	payload.Set("order_id", order.ID)

	b, err = trading.httpClient.Post(apiUserInfo, clients.Params(), payload, true)
	if err != nil {
		return
	}
	var resp OrdersCreateResponse
	err = json.Unmarshal(b, &resp)
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
