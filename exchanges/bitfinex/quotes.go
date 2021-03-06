package bitfinex

import (
	"sync"
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// QuotesProvider - quotes provider structure
type QuotesProvider struct {
	httpProxy proxy.Provider
	symbols   []schemas.Symbol
	groups    []*QuotesGroup

	sync.Mutex
}

// NewQuotesProvider - QuotesProvider constructor
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

// Get - getting quotes by symbol
func (qp *QuotesProvider) Get(symbol schemas.Symbol) (q schemas.Quote, err error) {
	group := NewQuotesGroup([]schemas.Symbol{symbol}, qp.httpProxy)
	return group.Get()
}

// Subscribe - subscribing to quote by one symbol
func (qp *QuotesProvider) Subscribe(symbol schemas.Symbol, d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)
	group := NewQuotesGroup([]schemas.Symbol{symbol}, qp.httpProxy)
	go group.Start(ch)
	return ch
}

// SubscribeAll - subscribing to all quotes with interval
func (qp *QuotesProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)

	for _, group := range qp.groups {
		go group.Start(ch)
		time.Sleep(100 * time.Millisecond)
	}
	return ch
}
