package poloniex

import (
	"encoding/json"
	"strconv"
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
			time.Sleep(interval)
		}
	}()

	go func() {
		for {
			uo, err := trading.Orders([]schemas.Symbol{})
			uoc <- schemas.UserOrdersChannel{
				Data:  uo,
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

	return trading.ordersBySymbol("all")
}

func (trading *TradingProvider) Trades(opts schemas.FilterOptions) (trades []schemas.Trade, p schemas.Paging, err error) {
	return
}

func (trading *TradingProvider) ImportTrades(opts schemas.FilterOptions) chan schemas.UserTradesChannel {
	ch := make(chan schemas.UserTradesChannel)
	return ch
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

func (trading *TradingProvider) ordersBySymbol(symbol string) (orders []schemas.Order, err error) {
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
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	for symbol, ords := range resp {
		for _, ord := range ords {
			orders = append(orders, ord.Map(symbol))
		}
	}

	return
}
