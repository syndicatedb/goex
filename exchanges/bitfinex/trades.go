package bitfinex

import (
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type TradesProvider struct {
	groups    []*TradesGroup
	httpProxy proxy.Provider
}

func NewTradesProvider(httpProxy proxy.Provider) *TradesProvider {
	return &TradesProvider{
		httpProxy: httpProxy,
	}
}

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

// TODO: add Get method here
func (tp *TradesProvider) Get(symbol schemas.Symbol) (q []schemas.Trade, err error) {
	return
}

// TODO: add subscribe to symbols method
func (tp *TradesProvider) Subscribe(symbol schemas.Symbol, d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)
	return ch
}

func (tp *TradesProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)

	for _, group := range tp.groups {
		go group.start(ch)
		time.Sleep(100 * time.Millisecond)
	}
	return ch
}
