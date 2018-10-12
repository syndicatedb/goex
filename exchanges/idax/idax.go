package idax

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/proxy"
	"github.com/syndicatedb/goex/schemas"
)

const (
	// URL - API endpoint
	apiSymbols     = "/api/v1/marketinfo"
	apiQuotes      = "/api/v1/tickers"
	apiQuote       = "/api/v1/ticker"
	apiOrderBook   = "/api/v1/depth/"
	apiTrades      = "/api/3/trades/"
	apiPrices      = "/api/v2/ticker"
	apiBalances    = "/api/v1/balances"
	apiOrderCreate = "/api/v1/createorder"
	apiOrderCancel = "/api/v1/cancelorder"
	apiUserOrders  = "/api/v1/myOrders"
	// apiUserTrades  = "/api/v1/myTrades"
	apiUserTrades = "/api/v2/tradesHistory"
)

const (
	// SubscriptionInterval - default subscription interval
	SubscriptionInterval  = 1 * time.Second
	orderBookSymbolsLimit = 10
	tradesSymbolsLimit    = 10
	quotesSymbolsLimit    = 10
)

var exchangeName = ""
var apiHost = "https://openapi.idax.mn"

/*
IDAX - exchange struct
*/
type IDAX struct {
	schemas.Exchange
}

// New - IDAX constructor. APIKey and APISecret is mandatory, but could be empty
func New(opts schemas.Options) *IDAX {
	exchangeName = opts.Name
	proxyProvider := opts.ProxyProvider
	if proxyProvider == nil {
		proxyProvider = proxy.NewNoProxy()
	}
	opts.Credentials.Sign = sign
	if opts.API != "" {
		apiHost = opts.API
	}
	return &IDAX{
		Exchange: schemas.Exchange{
			Credentials:   opts.Credentials,
			ProxyProvider: proxyProvider,
			Symbol:        NewSymbolsProvider(proxyProvider),
			Orders:        NewOrdersProvider(proxyProvider),
			Quotes:        NewQuotesProvider(proxyProvider),
			Trades:        NewTradesProvider(proxyProvider),
			Candles:       NewCandlesProvider(proxyProvider),
			Trading:       NewTradingProvider(opts.Credentials, proxyProvider),
		},
	}
}

func parseSymbol(s string) (name, coin, baseCoin string) {
	sa := strings.Split(s, "_")
	coin = strings.ToUpper(sa[0])
	baseCoin = strings.ToUpper(sa[1])
	name = coin + "-" + baseCoin
	return
}

func symbolToPair(s string) string {
	sa := strings.Split(s, "-")
	coin := sa[0]
	baseCoin := sa[1]
	return coin + "_" + baseCoin

}

// sign - signing request
func sign(key, secret string, req *http.Request) *http.Request {
	var query []string
	mts := time.Now().UTC().UnixNano() / 1000000
	timestamp := fmt.Sprintf("%d", mts)

	b, _ := req.GetBody()
	body, _ := ioutil.ReadAll(b)
	log.Printf("body: %+v\n", string(body))
	rawParams := make(map[string]string)
	for k, v := range req.URL.Query() {
		rawParams[k] = strings.Join(v, "")
	}
	rawParams["key"] = key
	rawParams["timestamp"] = timestamp
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

	// q := req.URL.Query()
	req.URL.Query().Add("key", key)
	req.URL.Query().Add("timestamp", timestamp)
	req.URL.Query().Add("sign", hex.EncodeToString(mac.Sum(nil)))
	req.URL.RawQuery = req.URL.Query().Encode()
	log.Println("req.URL.RawQuery: ", req.URL.RawQuery)
	return req
}

func getURL(url string) string {
	return apiHost + url
}

func getOrderSideByType(t string) string {
	if t == "BUY" {
		return "1"
	}
	return "2"
}

func getOrderTypeBySide(side int) string {
	if side == 1 {
		return "BUY"
	}
	return "SELL"
}
