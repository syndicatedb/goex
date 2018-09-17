package kucoin

import (
	"crypto/hmac"
	"crypto/sha256"
	b64 "encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/proxy"
	"github.com/syndicatedb/goex/schemas"
)

const (
	exchangeName = "kucoin"

	apiHost      = "https://api.kucoin.com"
	apiSymbols   = "https://api.kucoin.com/v1/market/open/symbols"
	apiCoins     = "https://api.kucoin.com/v1/market/open/coins"
	apiOrderBook = "https://api.kucoin.com/v1/open/orders"
	apiTrades    = "https://api.kucoin.com/v1/open/deal-orders"
	apiTicker    = "https://api.kucoin.com/v1/open/tick"

	apiUserBalance  = "https://api.kucoin.com/v1/account/balance"
	apiActiveOrders = "https://api.kucoin.com/v1/order/active-map"
	apiUserTrades   = "https://api.kucoin.com/v1/order/dealt"
	apiCandles      = "https://api.kucoin.com/v1/open/chart/history"

	apiCreateOrder = "https://api.kucoin.com/v1/order"
	apiCancelOrder = "https://api.kucoin.com/v1/cancel-order"
)

const (
	// SubscriptionInterval - default subscription interval
	SubscriptionInterval  = 1 * time.Second
	orderBookSymbolsLimit = 10
	tradesSymbolsLimit    = 10
	quotesSymbolsLimit    = 10
	defaultPrecision      = 4
)

// Kucoin - kucoin exchange structure
type Kucoin struct {
	schemas.Exchange
}

// New - Kucoin constructor
func New(opts schemas.Options) *Kucoin {
	proxyProvider := opts.ProxyProvider
	if proxyProvider == nil {
		proxyProvider = proxy.NewNoProxy()
	}
	opts.Credentials.Sign = sign
	return &Kucoin{
		Exchange: schemas.Exchange{
			Credentials:   opts.Credentials,
			ProxyProvider: proxyProvider,
			Symbol:        NewSymbolsProvider(proxyProvider),
			Orders:        NewOrdersProvider(proxyProvider),
			Trades:        NewTradesProvider(proxyProvider),
			Quotes:        NewQuotesProvider(proxyProvider),
			Candles:       NewCandlesProvider(proxyProvider),
			Trading:       NewTradingProvider(opts.Credentials, proxyProvider),
		},
	}
}

func parseSymbol(s string) (name, coin, baseCoin string) {
	sa := strings.Split(s, "-")
	coin = strings.ToUpper(sa[0])
	baseCoin = strings.ToUpper(sa[1])
	name = coin + "-" + baseCoin

	return
}

// sign - signing request
func sign(key, secret string, req *http.Request) *http.Request {
	// b, _ := req.GetBody()
	// body, err := ioutil.ReadAll(b)
	// if err != nil {
	// 	return req
	// }

	// log.Printf("req: %+v\n", req.URL.Path)
	// log.Printf("req: %+v\n", req.URL.Query().Encode())
	// log.Println("path: ", req.URL.String())
	path := req.URL.Path
	// path := req.URL.String()
	nonce := fmt.Sprintf("%v", time.Now().UnixNano()/int64(time.Millisecond))
	var signed string
	strForSign := path + "/" + nonce + "/" + req.URL.Query().Encode()
	// log.Println("strForSign: ", strForSign)
	signed = signRequest(strForSign, secret)
	req.Header.Add("KC-API-NONCE", nonce)
	req.Header.Add("KC-API-KEY", key)
	req.Header.Add("KC-API-SIGNATURE", signed)

	return req
}

func signRequest(str, secret string) string {

	// fmt.Println("strForSign", strForSign)
	signatureStr := b64.StdEncoding.EncodeToString([]byte(str))
	return computeHmac256(signatureStr, secret)
}

func computeHmac256(message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
