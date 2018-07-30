package bitfinex

import (
	"strings"

	"github.com/syndicatedb/goex/internal/proxy"
	"github.com/syndicatedb/goex/schemas"
)

const (
	exchangeName = "bitfinex"
	apiSymbols   = "https://api.bitfinex.com/v1/symbols_details"
)

type Bitfinex struct {
	schemas.Exchange
}

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
			// Orders:        NewOrdersProvider(proxyProvider),
			// Quotes:        NewQuotesProvider(proxyProvider),
			// Trades:        NewTradesProvider(proxyProvider),
			// Trading:       NewTradingProvider(opts.Credentials, proxyProvider),
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
