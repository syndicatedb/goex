package tidex

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/proxy"
	"github.com/syndicatedb/goex/schemas"
)

const (
	// URL - API endpoint
	apiSymbols   = "https://api.tidex.com/api/3/info"
	apiOrderBook = "https://api.tidex.com/api/3/depth/"
	apiQuotes    = "https://api.tidex.com/api/3/ticker/"
	apiTrades    = "https://api.tidex.com/api/3/trades/"
	apiUserInfo  = "https://api.tidex.com/tapi"
)

const (
	// SubscriptionInterval - default subscription interval
	SubscriptionInterval  = 1 * time.Second
	orderBookSymbolsLimit = 10
	tradesSymbolsLimit    = 10
	quotesSymbolsLimit    = 10
)

var exchangeName = ""

/*
Tidex - exchange struct
*/
type Tidex struct {
	schemas.Exchange
}

// New - Tidex constructor. APIKey and APISecret is mandatory, but could be empty
func New(opts schemas.Options) *Tidex {
	exchangeName = opts.Name
	proxyProvider := opts.ProxyProvider
	if proxyProvider == nil {
		proxyProvider = proxy.NewNoProxy()
	}
	opts.Credentials.Sign = sign
	tidex := &Tidex{
		Exchange: schemas.Exchange{
			Credentials:   opts.Credentials,
			ProxyProvider: proxyProvider,
			Symbol:        NewSymbolsProvider(proxyProvider),
			Orders:        NewOrdersProvider(proxyProvider),
			Quotes:        NewQuotesProvider(proxyProvider),
			Trades:        NewTradesProvider(proxyProvider),
			Candles:       NewCandlesProvider(proxyProvider),
		},
	}
	symbols, err := tidex.SymbolProvider().Get()
	if err != nil {
		log.Println("Error getting symbols", err)
	}
	tidex.Trading = NewTradingProvider(opts.Credentials, proxyProvider).SetSymbols(symbols)
	return tidex
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
	coin := strings.ToLower(sa[0])
	baseCoin := strings.ToLower(sa[1])
	return coin + "_" + baseCoin

}

// sign - signing request
func sign(key, secret string, req *http.Request) *http.Request {
	b, _ := req.GetBody()
	body, _ := ioutil.ReadAll(b)

	sig := hmac.New(sha512.New, []byte(secret))
	sig.Write(body)

	req.Header.Set("Sign", hex.EncodeToString(sig.Sum(nil)))
	req.Header.Set("Key", key)
	return req
}
