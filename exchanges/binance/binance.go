package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/proxy"
	"github.com/syndicatedb/goex/schemas"
)

const (
	exchangeName = "binance"
	apiSymbols   = "https://api.binance.com/api/v1/exchangeInfo"
	apiKlines    = "https://api.binance.com/api/v1/klines"
	apiOrderBook = "https://api.binance.com/api/v1/depth"
	apiTrades    = "https://api.binance.com/api/v1/trades"
	apiQuotes    = "https://api.binance.com/api/v1/ticker/24hr"

	apiUserBalance  = "https://api.binance.com/api/v3/account"
	apiActiveOrders = "https://api.binance.com/api/v3/openOrders"
	apiUserTrades   = "https://api.binance.com/api/v3/myTrades"

	apiCreateOrder = "https://api.binance.com/api/v3/order"
	apiCancelOrder = "https://api.binance.com/api/v3/order"

	wsURL = "wss://stream.binance.com:9443/stream?streams="
)

const (
	subscriptionInterval  = 1 * time.Second
	orderBookSymbolsLimit = 100
	tradesSymbolsLimit    = 10
	quotesSymbolsLimit    = 10
)

// Binance exchange structure
type Binance struct {
	schemas.Exchange
}

// New - bitfinex exchange constructor
func New(opts schemas.Options) *Binance {
	proxyProvider := opts.ProxyProvider
	if proxyProvider == nil {
		proxyProvider = proxy.NewNoProxy()
	}
	opts.Credentials.Sign = sign
	binance := &Binance{
		Exchange: schemas.Exchange{
			Credentials:   opts.Credentials,
			ProxyProvider: proxyProvider,
			Symbol:        NewSymbolsProvider(proxyProvider),
			Orders:        NewOrdersProvider(proxyProvider),
			Trades:        NewTradesProvider(proxyProvider),
			Quotes:        NewQuotesProvider(proxyProvider),
			Candles:       NewCandlesProvider(proxyProvider),
		},
	}
	symbols, err := binance.SymbolProvider().Get()
	if err != nil {
		log.Println("Error getting symbols", err)
	}
	binance.Trading = NewTradingProvider(opts.Credentials, proxyProvider, symbols)
	return binance
}

func parseSymbol(s string) (name, basecoin, quoteCoin string) {
	baseSymbols := []string{"USDT", "BTC", "ETH", "BNB"}

	for _, symb := range baseSymbols {
		if strings.Contains(s, symb) {
			if strings.LastIndex(s, symb)+len(symb) == len(s) {
				quoteCoin = strings.ToUpper(symb)
				basecoin = strings.ToUpper((strings.Replace(s, symb, "", -1)))
			}
		}
	}
	name = basecoin + "-" + quoteCoin
	return
}

func unparseSymbol(s string) (symbol string) {
	return strings.Replace(s, "-", "", 1)
}

// sign - signing request
func sign(key, secret string, req *http.Request) *http.Request {
	req.Header.Set("X-MBX-APIKEY", key)

	if req.URL.String() != httpURL {
		sign := createSignature256(req.URL.RawQuery, secret)
		// q := req.URL.Query()
		// q.Add("signature", sign)
		req.URL.RawQuery += "&signature=" + sign
	}
	return req
}

func createSignature256(query, secretKey string) (signature string) {
	hash := hmac.New(sha256.New, []byte(secretKey))
	hash.Write([]byte(query))
	signature = hex.EncodeToString(hash.Sum(nil))
	return
}
