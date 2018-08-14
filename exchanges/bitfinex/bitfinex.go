package bitfinex

import (
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/proxy"
	"github.com/syndicatedb/goex/schemas"
)

const (
	exchangeName = "bitfinex"
	apiSymbols   = "https://api.bitfinex.com/v1/symbols_details"
	apiOrderBook = "https://api.bitfinex.com/v2/book"
	apiTrades    = "https://api.bitfinex.com/v2/trades"
	apiQuotes    = "https://api.bitfinex.com/v2/ticker"
	apiCandles   = "https://api.bitfinex.com/v2/candles"

	wsURL = "wss://api.bitfinex.com/ws/2"
)

const (
	subscriptionInterval  = 1 * time.Second
	orderBookSymbolsLimit = 100
	tradesSymbolsLimit    = 10
	quotesSymbolsLimit    = 10
)

// Bitfinex - bitfinex exchange structure
type Bitfinex struct {
	schemas.Exchange
}

// New - bitfinex exchange constructor
func New(opts schemas.Options) *Bitfinex {
	proxyProvider := opts.ProxyProvider
	if proxyProvider == nil {
		proxyProvider = proxy.NewNoProxy()
	}

	return &Bitfinex{
		Exchange: schemas.Exchange{
			Credentials:   opts.Credentials,
			ProxyProvider: proxyProvider,
			Symbol:        NewSymbolsProvider(proxyProvider),
			Orders:        NewOrdersProvider(proxyProvider),
			Trades:        NewTradesProvider(proxyProvider),
			Quotes:        NewQuotesProvider(proxyProvider),
			Candles:       NewCandlesProvider(proxyProvider),
			// Trading:       NewTradingProvider(opts.Credentials, proxyProvider),
		},
	}
}

func parseSymbol(smb string) (name, basecoin, quoteCoin string) {
	if strings.Index(smb, "t") == 0 {
		smb = strings.Replace(smb, "t", "", 1)
	}
	s := strings.ToUpper(smb)
	baseSymbols := []string{"USD", "EUR", "GBP", "JPY", "BTC", "ETH", "EOS"}

	for _, symb := range baseSymbols {
		if strings.Contains(s, symb) {
			if strings.LastIndex(s, symb)+2 == len(s)-1 {
				quoteCoin = strings.ToUpper(symb)
				basecoin = strings.ToUpper((strings.Replace(s, symb, "", -1)))
			}
		}

	}
	name = basecoin + "-" + quoteCoin
	return
}
