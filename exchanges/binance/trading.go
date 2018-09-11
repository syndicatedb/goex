package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	var b []byte
	// params := httpclient.Params()

	query := "timestamp=" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)[:13]
	signature := createSignature256(query, trading.credentials.APISecret)

	url := apiUserBalance + "?" + query + "&" + signature

	b, err = trading.httpClient.Get(url, httpclient.Params(), false)
	if err != nil {
		return
	}
	var resp UserBalanceResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	return resp.Map(), nil
}

func createSignature256(query, secretKey string) (signature string) {
	hash := hmac.New(sha256.New, []byte(secretKey))
	hash.Write([]byte(query))
	signature = hex.EncodeToString(hash.Sum(nil))
	return
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
		interval = time.Second
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
	var b []byte

	query := "symbol=" + "&timestamp=" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)[:13]
	signature := createSignature256(query, trading.credentials.APISecret)

	url := apiActiveOrders + "?" + query + "&" + signature

	b, err = trading.httpClient.Get(url, httpclient.Params(), false)
	if err != nil {
		return
	}
	var resp UserOrdersResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}

	return resp.Map(), nil
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

	query := "symbol=" + "&timestamp=" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)[:13]
	signature := createSignature256(query, trading.credentials.APISecret)

	url := apiUserTrades + "?" + query + "&" + signature

	b, err = trading.httpClient.Get(url, params, false)
	if err != nil {
		return
	}
	var resp UserTradesResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}

	return resp.Map(), schemas.Paging{}, nil
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
