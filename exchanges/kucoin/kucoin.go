package kucoin

import (
	"github.com/syndicatedb/goex/internal/proxy"
	"github.com/syndicatedb/goex/schemas"
)

const (
	exchangeName = "kucoin"

	apiSymbols = "https://api.kucoin.com/v1/market/open/symbols"
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
			// Orders:        NewOrdersProvider(proxyProvider),
			// Quotes:        NewQuotesProvider(proxyProvider),
			// Trades:        NewTradesProvider(proxyProvider),
			// Trading:       NewTradingProvider(opts.Credentials, proxyProvider),
		},
	}
}

// TODO: parseSymbol function
func parseSymbol(s string) (name, coin, baseCoin string) {
	return
}
