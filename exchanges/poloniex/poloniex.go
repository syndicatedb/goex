package poloniex

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
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
	exchangeName = "poloniex"

	restURL    = "https://poloniex.com/public"
	wsURL      = "wss://api2.poloniex.com"
	tradingAPI = "https://poloniex.com/tradingApi"
)

const (
	subscriptionInterval  = 1 * time.Second
	snapshotInterval      = 5 * time.Minute
	orderBookSymbolsLimit = 300
	tradesSymbolsLimit    = 10
	quotesSymbolsLimit    = 10
	defaultPrecision      = 8

	commandSubscribe        = "subscribe"
	commandCompleteBalances = "returnCompleteBalances"
	commandOrderBook        = "returnOrderBook"
	commandVolumes          = "return24hVolume"
	commandTicker           = "returnTicker"
	commandOpenOrders       = "returnOpenOrders"
	commandTrades           = "returnTradeHistory"

	commandBalance = "returnCompleteBalances"
)

// Poloniex - poloniex exchange structure
type Poloniex struct {
	schemas.Exchange
}

// New - poloniex exchange constructor
func New(opts schemas.Options) *Poloniex {
	proxyProvider := opts.ProxyProvider
	if proxyProvider == nil {
		proxyProvider = proxy.NewNoProxy()
	}

	opts.Credentials.Sign = sign
	return &Poloniex{
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

func parseSymbol(s string) (name, basecoin, quoteCoin string) {
	sa := strings.Split(s, "_")
	basecoin = sa[1]
	quoteCoin = sa[0]
	name = basecoin + "-" + quoteCoin

	return
}

func sign(key, secret string, req *http.Request) *http.Request {
	var signed string

	b, _ := req.GetBody()
	body, err := ioutil.ReadAll(b)
	if err != nil {
		return req
	}
	log.Printf("BODY %+v", string(body))

	// nonce := fmt.Sprintf("%v", time.Now().UnixNano()/int64(time.Millisecond))
	signed = signRequest(string(body), secret)
	req.Header.Set("Key", key)
	req.Header.Set("Sign", signed)
	return req
}

func signRequest(str, secret string) string {
	signatureStr := base64.StdEncoding.EncodeToString([]byte(str))
	return computeHmac512(signatureStr, secret)
}

func computeHmac512(message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha512.New, key)
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
