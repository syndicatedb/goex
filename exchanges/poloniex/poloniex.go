package poloniex

import (
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/proxy"

	"github.com/syndicatedb/goex/schemas"
)

const (
	exchangeName = "poloniex"

	restURL = "https://poloniex.com/public"
	wsURL   = "wss://api2.poloniex.com/realm1"
)

const (
	subscriptionInterval  = 1 * time.Second
	orderBookSymbolsLimit = 100
	tradesSymbolsLimit    = 10
	quotesSymbolsLimit    = 10

	commandSubscribe        = "subscribe"
	commandCompleteBalances = "returnCompleteBalances"
	commandOrderBook        = "returnOrderBook"
	commandVolumes          = "return24hVolume"
	commandOpenOrders       = "returnOpenOrders"

	reconnectAttemps   = 5
	resubscribeAttemps = 5
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

	return &Poloniex{
		Exchange: schemas.Exchange{
			Credentials:   opts.Credentials,
			ProxyProvider: proxyProvider,
			Symbol:        NewSymbolsProvider(proxyProvider),
			Orders:        NewOrdersProvider(proxyProvider),
			//Trades:        NewTradesProvider(proxyProvider),
			//Quotes:        NewQuotesProvider(proxyProvider),
			// Trading:       NewTradingProvider(opts.Credentials, proxyProvider),
		},
	}
}

func parseSymbol(s string) (name, basecoin, quoteCoin string) {
	sa := strings.Split(s, "_")
	basecoin = sa[1]
	quoteCoin = sa[0]
	name = basecoin + "_" + quoteCoin

	return
}
