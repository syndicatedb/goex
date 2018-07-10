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

// UserProvider - provides quotes/ticker
type UserProvider struct {
	credentials schemas.Credentials
	httpProxy   proxy.Provider
	httpClient  *clients.HTTP
}

// NewUserProvider - UserProvider constructor
func NewUserProvider(credentials schemas.Credentials, httpProxy proxy.Provider) *UserProvider {
	proxyClient := httpProxy.NewClient(exchangeName)
	return &UserProvider{
		credentials: credentials,
		httpProxy:   httpProxy,
		httpClient:  clients.NewSignedHTTP(credentials, proxyClient),
	}
}

// Info - provides user info: Keys access, balances
func (up *UserProvider) Info() (ui schemas.UserInfo, err error) {
	var b []byte
	payload := clients.Params()
	payload.Set("method", "getInfoExt")
	payload.Set("nonce", fmt.Sprintf("%d", time.Now().Unix()))

	b, err = up.httpClient.Post(apiUserInfo, clients.Params(), payload, true)
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
func (up *UserProvider) Subscribe(interval time.Duration) (chan schemas.UserInfoChannel, chan schemas.UserOrdersChannel, chan schemas.UserTradesChannel) {
	uic := make(chan schemas.UserInfoChannel)
	uoc := make(chan schemas.UserOrdersChannel)
	utc := make(chan schemas.UserTradesChannel)

	if interval == 0 {
		interval = SubscriptionInterval
	}
	lastTradeID := "1"
	go func() {
		for {
			ui, err := up.Info()
			uic <- schemas.UserInfoChannel{
				Data:  ui,
				Error: err,
			}
			o, err := up.Orders([]schemas.Symbol{})
			uoc <- schemas.UserOrdersChannel{
				Data:  o,
				Error: err,
			}
			t, err := up.Trades(schemas.TradeHistoryOptions{
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
func (up *UserProvider) Orders(symbols []schemas.Symbol) (orders []schemas.Order, err error) {
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
	b, err = up.httpClient.Post(apiUserInfo, clients.Params(), payload, true)
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
func (up *UserProvider) Trades(opts schemas.TradeHistoryOptions) (trades []schemas.Trade, err error) {
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

	b, err = up.httpClient.Post(apiUserInfo, clients.Params(), payload, true)
	if err != nil {
		return
	}
	var resp UserTradesResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	return resp.Map(), nil
}
