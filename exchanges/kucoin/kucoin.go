package kucoin

import (
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/proxy"
	"github.com/syndicatedb/goex/schemas"
)

const (
	exchangeName = "kucoin"

	apiSymbols   = "https://api.kucoin.com/v1/market/open/symbols"
	apiOrderBook = "https://api.kucoin.com/v1/open/orders"
	apiTrades    = "https://api.kucoin.com/v1/open/deal-orders"
)

const (
	// SubscriptionInterval - default subscription interval
	SubscriptionInterval  = 1 * time.Second
	orderBookSymbolsLimit = 10
	tradesSymbolsLimit    = 10
	quotesSymbolsLimit    = 10
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
	return &Kucoin{
		Exchange: schemas.Exchange{
			Credentials:   opts.Credentials,
			ProxyProvider: proxyProvider,
			Symbol:        NewSymbolsProvider(proxyProvider),
			Orders:        NewOrdersProvider(proxyProvider),
			Trades:        NewTradesProvider(proxyProvider),
			// Quotes:        NewQuotesProvider(proxyProvider),
			// Trading:       NewTradingProvider(opts.Credentials, proxyProvider),
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
