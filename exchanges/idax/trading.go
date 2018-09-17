package idax

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// TradingProvider - provides quotes/ticker
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

// Info - provides user info: Keys access, balances
func (trading *TradingProvider) Info() (ui schemas.UserInfo, err error) {
	ui.Balances, err = trading.Balances()
	return
}

// Balances user balances by Coin
func (trading *TradingProvider) Balances() (balances map[string]schemas.Balance, err error) {
	var b []byte
	balances = make(map[string]schemas.Balance)

	emptyParams := httpclient.Params()
	if b, err = trading.httpClient.Get(getURL(apiBalances), emptyParams, true); err != nil {
		return
	}
	var resp Response
	if err = json.Unmarshal(b, &resp); err != nil {
		fmt.Println("[IDAX] Response error:", string(b))
		return
	}
	if resp.Success != true {
		log.Println("[IDAX] Error in Balance response: ", resp.Message)
		err = errors.New(resp.Message)
		return
	}
	var items []Balance
	if err = json.Unmarshal(resp.Data, &items); err != nil {
		fmt.Println("[IDAX] Balance parsing error:", err, string(b))
		return
	}
	for _, b := range items {
		balances[b.CoinCode] = b.Map()
	}
	return
}

/*
Subscribe - subscribing to user info
— user info
- orders
- trades
*/
func (trading *TradingProvider) Subscribe(interval time.Duration) (chan schemas.UserInfoChannel, chan schemas.UserOrdersChannel, chan schemas.UserTradesChannel) {
	uic := make(chan schemas.UserInfoChannel, 300)
	uoc := make(chan schemas.UserOrdersChannel, 300)
	utc := make(chan schemas.UserTradesChannel, 300)

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
			t, _, err := trading.Trades(schemas.FilterOptions{
				FromID: lastTradeID,
				Limit:  200,
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
	params := httpclient.Params()
	params.Set("top", "100")

	b, err = trading.httpClient.Get(getURL(apiUserOrders), params, true)
	if err != nil {
		return
	}
	var resp Response
	if err = json.Unmarshal(b, &resp); err != nil {
		log.Println("[IDAX] Error getting user orders: ", err)
		return
	}
	if resp.Success != true {
		err = errors.New(resp.Message)
		return
	}
	var userOrders []UserOrder
	if err = json.Unmarshal(resp.Data, &userOrders); err != nil {
		return
	}
	for _, o := range userOrders {
		orders = append(orders, o.Map())
	}
	return
}

// ImportTrades - importing trades
func (trading *TradingProvider) ImportTrades(opts schemas.FilterOptions) chan schemas.UserTradesChannel {
	ch := make(chan schemas.UserTradesChannel)

	trades, paging, err := trading.Trades(opts)
	log.Println("paging: ", len(trades), paging, err)

	return ch
}

// Trades - getting user trades
func (trading *TradingProvider) Trades(opts schemas.FilterOptions) (trades []schemas.Trade, p schemas.Paging, err error) {
	var b []byte
	payload := httpclient.Params()
	payload.Set("method", "TradeHistory")
	payload.Set("nonce", fmt.Sprintf("%d", time.Now().Unix()))

	if len(opts.Symbols) > 0 {
		var pairs []string
		for _, s := range opts.Symbols {
			pairs = append(pairs, symbolToPair(s.Name))
		}
		payload.Set("pair", strings.Join(pairs, "-"))
	}

	if opts.Limit > 0 {
		payload.Set("count", fmt.Sprintf("%d", opts.Limit))
	}

	if opts.FromID != "" {
		payload.Set("from_id", opts.FromID)
	}
	b, err = trading.httpClient.Post(getURL(apiUserTrades), httpclient.Params(), payload, true)
	if err != nil {
		return
	}
	var resp UserTradesResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	return resp.Map(), p, nil
}

// Create - creating order
func (trading *TradingProvider) Create(order schemas.Order) (result schemas.Order, err error) {
	var b []byte

	payload := httpclient.Params()

	params := httpclient.Params()

	price := strconv.FormatFloat(order.Price, 'f', -1, 64)
	amount := strconv.FormatFloat(order.Amount, 'f', -1, 64)

	params.Set("orderSide", getOrderSideByType(order.Type))
	params.Set("orderType", "1")
	params.Set("pair", symbolToPair(order.Symbol))
	params.Set("price", price)
	params.Set("amount", amount)

	b, err = trading.httpClient.Post(getURL(apiOrderCreate), params, payload, true)
	if err != nil {
		return
	}
	var resp Response
	if err = json.Unmarshal(b, &resp); err != nil {
		log.Println("[IDAX] Error create order: ", err)
		return
	}
	if resp.Success != true {
		log.Println("[IDAX] Create resp.Message: ", resp.Message)
		err = errors.New(resp.Message)
		return
	}
	if b, err = json.Marshal(&resp.Data); err != nil {
		return
	}
	order.ID = string(b)
	order.CreatedAt = time.Now().UTC().UnixNano() / 1000000
	return order, nil
}

// Cancel - cancelling order
func (trading *TradingProvider) Cancel(order schemas.Order) (err error) {
	var b []byte

	params := httpclient.Params()
	payload := httpclient.Params()

	params.Set("orderId", order.ID)

	b, err = trading.httpClient.Post(getURL(apiOrderCancel), params, payload, true)
	if err != nil {
		return
	}
	var resp Response
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if resp.Success != true {
		err = errors.New(resp.Message)
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
