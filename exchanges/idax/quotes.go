package idax

import (
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// QuotesProvider - provides quotes/ticker
type QuotesProvider struct {
	symbols   []schemas.Symbol
	groups    []*QuotesGroup
	httpProxy proxy.Provider
}

// NewQuotesProvider - QuotesProvider constructor
func NewQuotesProvider(httpProxy proxy.Provider) *QuotesProvider {
	return &QuotesProvider{
		httpProxy: httpProxy,
	}
}

// SetSymbols - getting all symbols from Exchange
func (qp *QuotesProvider) SetSymbols(symbols []schemas.Symbol) schemas.QuotesProvider {
	qp.symbols = symbols
	slice := make([]schemas.Symbol, len(symbols))
	copy(slice, symbols)
	capacity := quotesSymbolsLimit
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
	var data []schemas.Quote
	group := NewQuotesGroup([]schemas.Symbol{symbol}, qp.httpProxy)
	data, err = group.Get()
	if err != nil {
		return
	}
	return data[0], nil
}

// Subscribe - subscribing to quote by symbol and interval
func (qp *QuotesProvider) Subscribe(symbol schemas.Symbol, d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)
	group := NewQuotesGroup([]schemas.Symbol{symbol}, qp.httpProxy)
	go group.subscribe(ch, d)
	return ch
}

// SubscribeAll - subscribing to all quotes with interval
func (qp *QuotesProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	bufLength := 2 * len(qp.symbols)
	ch := make(chan schemas.ResultChannel, bufLength)

	for _, group := range qp.groups {
		go group.subscribe(ch, d)
		time.Sleep(100 * time.Millisecond)
	}
	return ch
}
