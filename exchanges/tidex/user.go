package tidex

import (
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
		httpClient:  clients.NewHTTP(proxyClient),
	}
}

// Info - provides user info: Keys access, balances
func (up *UserProvider) Info() {
	payload := clients.Params()
	payload.Set("method", "getInfoExt")
	payload.Set("nonce", fmt.Sprintf("%d", time.Now().Unix()))
	secret := signRequest(up.credentials.APISecret, payload.Map())

	up.httpClient.Headers.Set("Sign", secret)
	up.httpClient.Headers.Set("Key", up.credentials.APIKey)

	b, err := up.httpClient.Post(apiUserInfo, clients.Params(), payload)

	fmt.Println("params: ", string(b), err)
}

/*
Subscribe - subscribing to user info
— keys state
- balances
- orders
- trades
*/
func (up *UserProvider) Subscribe() {}

func (up *UserProvider) Orders(symbol []schemas.Symbol) (orders []schemas.Order, err error) {
	return
}

func (up *UserProvider) Trades(symbol []schemas.Symbol) (orders []schemas.Order, err error) {
	return
}

func (up *UserProvider) TradesHistory(sinceTrade schemas.Trade) (trades []schemas.Trade, err error) {
	return
}
