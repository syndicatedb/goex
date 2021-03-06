package idax

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	httpclient "github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// TradingProvider - provides quotes/ticker
type TradingProvider struct {
	credentials schemas.Credentials
	httpProxy   proxy.Provider
	httpClient  *httpclient.Client
	symbols     []schemas.Symbol
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

// SetSymbols update symbols in trading provider
func (trading *TradingProvider) SetSymbols(symbols []schemas.Symbol) schemas.TradingProvider {
	trading.symbols = symbols

	return trading
}

// Info - provides user info: Keys access, balances
func (trading *TradingProvider) Info() (ui schemas.UserInfo, err error) {
	ui.Balances, err = trading.Balances()
	if err != nil {
		return
	}

	ui.Prices, err = trading.prices()

	return
}

type priceResponse struct {
	Code      int     `json:"code"`
	Msg       string  `json:"msg"`
	Timestamp int64   `json:"timestamp"`
	Ticker    []price `json:"ticker"`
}
type price struct {
	Symbol string `json:"pair"`
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Last   string `json:"last"`
	Volume string `json:"vol"`
}

func (trading *TradingProvider) prices() (resp map[string]float64, err error) {
	var b []byte

	b, err = trading.httpClient.Get(getURL(apiPrices), httpclient.Params(), false)
	if err != nil {
		return
	}

	var prices priceResponse
	if err = json.Unmarshal(b, &prices); err != nil {
		return
	}

	resp = make(map[string]float64)
	for _, p := range prices.Ticker {
		symbol, _, _ := parseSymbol(p.Symbol)
		price, err := strconv.ParseFloat(p.Last, 64)
		if err != nil {
			log.Println("Error parsing price for balances", err)
		}
		resp[symbol] = price
	}

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
	if len(opts.Symbols) == 0 {
		err = errors.New("Symbols empty")
	}
	for _, s := range opts.Symbols {
		var b []byte
		var req *http.Request
		payload := httpclient.Params()
		payload.Set("pair", symbolToPair(s.Name))
		payload.Set("since", "")
		if opts.FromID != "" {
			payload.Set("since", opts.FromID)
		}
		log.Printf("payload: %+v\n", payload)
		req, err = signJSON(trading.credentials.APIKey, trading.credentials.APISecret, getURL(apiUserTrades), payload)
		if err != nil {
			continue
		}
		b, err = trading.httpClient.Do(req)
		if err != nil {
			continue
		}
		var resp UserTradesResponse
		if err = json.Unmarshal(b, &resp); err != nil {
			continue
		}
		tr := resp.Map(s.Name)
		trades = append(trades, tr...)
	}
	return
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

func signJSON(key, secret, url string, payload httpclient.KeyValue) (*http.Request, error) {
	// pair and since already in payload
	var query []string
	mts := time.Now().UTC().UnixNano() / 1000000
	timestamp := fmt.Sprintf("%d", mts)

	payload.Set("key", key)
	payload.Set("timestamp", timestamp)
	rawParams := payload.Map()
	var keys []string
	for k := range rawParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		query = append(query, k+"="+rawParams[k])
	}
	str := strings.Join(query, "&")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(str))

	payload.Set("sign", hex.EncodeToString(mac.Sum(nil)))
	b, err := json.Marshal(payload.Map())
	if err != nil {
		log.Println("Marshalling error in sign json", err)
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		log.Println("Error creating new request in sign json", err)
		return nil, err
	}
	req.Header.Set("Content-Type", httpclient.ContentTypeJSON)

	return req, nil
}
