package poloniex

import (
	"sync"
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type TradesProvider struct {
	groups    []*TradesGroup
	httpProxy proxy.Provider

	sync.Mutex
}

func NewTradesProvider(httpProxy proxy.Provider) *TradesProvider {
	return &TradesProvider{
		httpProxy: httpProxy,
	}
}

// SetSymbols - setting symbols to TradesProvider
func (tp *TradesProvider) SetSymbols(symbols []schemas.Symbol) schemas.TradesProvider {
	slice := make([]schemas.Symbol, len(symbols))
	copy(slice, symbols)
	capacity := orderBookSymbolsLimit
	for {
		if len(slice) <= capacity {
			tp.groups = append(
				tp.groups,
				NewTradesGroup(slice, tp.httpProxy),
			)
			break
		}
		tp.groups = append(
			tp.groups,
			NewTradesGroup(slice[0:capacity], tp.httpProxy),
		)

		slice = slice[capacity:]
	}
	return tp
}

// Get - getting trades snapshot by symbol
func (tp *TradesProvider) Get(symbol schemas.Symbol) (q []schemas.Trade, err error) {
	group := NewTradesGroup([]schemas.Symbol{symbol}, tp.httpProxy)
	return group.Get()
}

// Subscribe - subscribing to trades by one symbol
func (tp *TradesProvider) Subscribe(symbol schemas.Symbol, d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)
	group := NewTradesGroup([]schemas.Symbol{symbol}, tp.httpProxy)
	go group.Start(ch)
	return ch
}

// SubscribeAll - subscribing all groups
func (tp *TradesProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)

	for _, group := range tp.groups {
		go group.Start(ch)
		time.Sleep(100 * time.Millisecond)
	}
	return ch
}
