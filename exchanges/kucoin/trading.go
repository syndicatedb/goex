package kucoin

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	var b []byte
	params := httpclient.Params()
	params.Set("coin", "")
	params.Set("nonce", fmt.Sprintf("%d", time.Now().Unix()))

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
			t, _, err := trading.Trades(schemas.FilterOptions{
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
	// var b []byte
	// payload := httpclient.Params()
	// payload.Set("method", "ActiveOrders")
	// payload.Set("nonce", fmt.Sprintf("%d", time.Now().Unix()))
	// if len(symbols) > 0 {
	// 	var pairs []string
	// 	for _, s := range symbols {
	// 		pairs = append(pairs, s.OriginalName)
	// 	}
	// 	payload.Set("pair", strings.Join(pairs, "-"))
	// }
	// b, err = trading.httpClient.Post(apiUserInfo, httpclient.Params(), payload, true)
	// if err != nil {
	// 	return
	// }
	// var resp UserOrdersResponse
	// if err = json.Unmarshal(b, &resp); err != nil {
	// 	return
	// }
	// return resp.Map(), nil
	return
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
	var b []byte
	params := httpclient.Params()

	if len(opts.Symbols) > 0 {
		var pairs []string
		for _, s := range opts.Symbols {
			pairs = append(pairs, s.Name)
		}
		params.Set("symbol", strings.Join(pairs, ","))
	}

	if opts.Limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", opts.Limit))
	}

	if opts.Since != 0 {
		params.Set("since", fmt.Sprintf("%d", opts.Since))
	}
	if opts.Before != 0 {
		params.Set("before", fmt.Sprintf("%d", opts.Before))
	}
	if opts.Page != 0 {
		params.Set("page", fmt.Sprintf("%d", opts.Page))
	}
	b, err = trading.httpClient.Get(apiUserTrades, params, true)
	if err != nil {
		return
	}
	var resp UserTradesResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if resp.Success == false {
		log.Printf("resp error: %+v\n", resp)
		if resp.Code == "UNAUTH" {
			err = errors.New(resp.Msg)
			return
		}
	}
	return resp.Map(), schemas.Paging{
		Count:   resp.Data.Total,
		Pages:   resp.Data.PageNos,
		Current: resp.Data.CurrPageNo,
		Limit:   resp.Data.Limit,
	}, nil
}

// Create - creating order
func (trading *TradingProvider) Create(order schemas.Order) (result schemas.Order, err error) {
	// 	var b []byte

	// 	payload := httpclient.Params()
	// 	payload.Set("method", "Trade")
	// 	payload.Set("nonce", fmt.Sprintf("%d", time.Now().Unix()))

	// 	pair := symbolToPair(order.Symbol)
	// 	payload.Set("pair", pair)
	// 	payload.Set("type", strings.ToLower(order.Type))
	// 	payload.Set("rate", fmt.Sprintf("%f", order.Price))
	// 	payload.Set("amount", fmt.Sprintf("%f", order.Amount))

	// 	b, err = trading.httpClient.Post(apiUserInfo, httpclient.Params(), payload, true)
	// 	if err != nil {
	// 		return
	// 	}
	// 	var resp OrdersCreateResponse
	// 	if err = json.Unmarshal(b, &resp); err != nil {
	// 		return
	// 	}
	// 	order.ID = fmt.Sprintf("%d", resp.Return.OrderID)
	// 	order.CreatedAt = time.Now().UTC().UnixNano() / 1000000
	// 	return order, nil
	return
}

// Cancel - cancelling order
func (trading *TradingProvider) Cancel(order schemas.Order) (err error) {
	// 	var b []byte

	// 	payload := httpclient.Params()
	// 	payload.Set("method", "CancelOrder")
	// 	payload.Set("nonce", fmt.Sprintf("%d", time.Now().Unix()))

	// 	payload.Set("order_id", order.ID)

	// 	b, err = trading.httpClient.Post(apiUserInfo, httpclient.Params(), payload, true)
	// 	if err != nil {
	// 		return
	// 	}
	// 	var resp OrdersCreateResponse
	// 	err = json.Unmarshal(b, &resp)
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
