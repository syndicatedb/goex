package tidex

import (
	"strings"

	"github.com/syndicatedb/goex/clients"
	"github.com/syndicatedb/goex/schemas"
)

const (
	// URL - API endpoint
	apiSymbols   = "https://api.tidex.com/api/3/info"
	apiOrderBook = "https://api.tidex.com/api/3/depth/"
	apiQuotes    = "https://api.tidex.com/api/3/ticker/"
	apiTrades    = "https://api.tidex.com/api/3/trades/"
)

var (
	orderBookSymbolsLimit = 20
	quotesSymbolsLimit    = 10
	exchangeName          = ""
)

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
		proxyProvider = clients.NewNoProxy()
	}
	return &Tidex{
		Exchange: schemas.Exchange{
			Credentials:    opts.Credentials,
			ProxyProvider:  proxyProvider,
			SymbolProvider: NewSymbolsProvider(proxyProvider),
			OrdersProvider: NewOrdersProvider(proxyProvider),
			QuotesProvider: NewQuotesProvider(proxyProvider),
			TradesProvider: NewTradesProvider(proxyProvider),
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
