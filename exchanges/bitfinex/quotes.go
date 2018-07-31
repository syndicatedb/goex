package bitfinex

import (
	"sync"
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type QuotesProvider struct {
	httpProxy proxy.Provider
	symbols   []schemas.Symbol
	groups    []*QuotesGroup

	sync.Mutex
}

func NewQuotesProvider(httpProxy proxy.Provider) *QuotesProvider {
	return &QuotesProvider{
		httpProxy: httpProxy,
	}
}

// SetSymbols - setting symbols and creating groups by symbols chunks
func (qp *QuotesProvider) SetSymbols(symbols []schemas.Symbol) schemas.QuotesProvider {
	slice := make([]schemas.Symbol, len(symbols))
	copy(slice, symbols)
	capacity := orderBookSymbolsLimit
	for {
		if len(slice) <= capacity {
			qp.groups = append(
				qp.groups,
				NewQuotesGroup(slice, qp.httpProxy),
			)
			break
		}
		qp.groups = append(
			qp.groups,
			NewQuotesGroup(slice[0:capacity], qp.httpProxy),
		)

		slice = slice[capacity:]
	}
	return qp
}

// TODO: Add get and subscribe methods
// Get - getting quotes by symbol
func (qp *QuotesProvider) Get(symbol schemas.Symbol) (q schemas.Quote, err error) {
	return
}

// Subscribe - subscribing to quote by symbol and interval
func (qp *QuotesProvider) Subscribe(symbol schemas.Symbol, d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)
	return ch
}

// SubscribeAll - subscribing to all quotes with interval
func (qp *QuotesProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)

	for _, group := range qp.groups {
		go group.start(ch)
		time.Sleep(100 * time.Millisecond)
	}
	return ch
}
