package tidex

import (
	"encoding/json"
	"fmt"
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
func (up *UserProvider) Subscribe(interval time.Duration) chan schemas.UserInfoChannel {
	uic := make(chan schemas.UserInfoChannel)

	if interval == 0 {
		interval = SubscriptionInterval
	}
	go func() {
		for {
			ui, err := up.Info()
			uic <- schemas.UserInfoChannel{
				Data:  ui,
				Error: err,
			}
			time.Sleep(interval)
		}
	}()
	return uic
}

func (up *UserProvider) Orders(symbol []schemas.Symbol) (orders []schemas.Order, err error) {
	return
}

func (up *UserProvider) Trades(sinceTrade schemas.Trade) (orders []schemas.Order, err error) {
	return
}
