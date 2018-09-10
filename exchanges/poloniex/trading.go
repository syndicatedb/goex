package poloniex

import (
	"encoding/json"
	"fmt"
	"log"
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

	go func() {
		for {
			ui, err := trading.Info()
			uic <- schemas.UserInfoChannel{
				Data:  ui,
				Error: err,
			}
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
	nonce := fmt.Sprintf("%v", time.Now().Unix())
	payload.Set("nonce", nonce)
	payload.Set("command", commandBalance)

	b, err = trading.httpClient.Post(tradingAPI, httpclient.Params(), payload, true)
	if err != nil {
		return
	}
	log.Printf("RESPONSE %+v", string(b))

	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	for coin, value := range resp {
		userBalance[coin] = value.Map(coin)
	}

	ui.Balances = userBalance
	return
}

func (trading *TradingProvider) Orders(symbols []schemas.Symbol) (orders []schemas.Order, err error) {
	return
}

func (trading *TradingProvider) ImportTrades(opts schemas.FilterOptions) chan schemas.UserTradesChannel {
	ch := make(chan schemas.UserTradesChannel)
	return ch
}

func (trading *TradingProvider) Trades(opts schemas.FilterOptions) (trades []schemas.Trade, p schemas.Paging, err error) {
	return
}

func (trading *TradingProvider) Create(order schemas.Order) (result schemas.Order, err error) {
	return
}

func (trading *TradingProvider) Cancel(order schemas.Order) (err error) {
	return
}

func (trading *TradingProvider) CancelAll() (err error) {
	return
}
