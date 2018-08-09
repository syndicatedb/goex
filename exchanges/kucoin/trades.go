package kucoin

import (
	"fmt"
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type TradesProvider struct {
	symbols   []schemas.Symbol
	groups    []*TradesGroup
	httpProxy proxy.Provider
}

// NewTradesProvider - TradesProvider constructor
func NewTradesProvider(httpProxy proxy.Provider) *TradesProvider {
	return &TradesProvider{
		httpProxy: httpProxy,
	}
}

// SetSymbols - getting all symbols from Exchange
func (tp *TradesProvider) SetSymbols(symbols []schemas.Symbol) schemas.TradesProvider {
	tp.symbols = symbols
	slice := make([]schemas.Symbol, len(symbols))
	copy(slice, symbols)
	capacity := tradesSymbolsLimit
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

// Get - getting quotes by symbol
func (tp *TradesProvider) Get(symbol schemas.Symbol) (q []schemas.Trade, err error) {
	var data [][]schemas.Trade
	group := NewTradesGroup([]schemas.Symbol{symbol}, tp.httpProxy)
	data, err = group.Get()
	if err != nil {
		return
	}
	if len(data) > 0 {
		return data[0], nil
	}

	err = fmt.Errorf("No trades found for %s", symbol.Name)
	return
}

// Subscribe - subscribing to quote by symbol and interval
func (tp *TradesProvider) Subscribe(symbol schemas.Symbol, d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)
	group := NewTradesGroup([]schemas.Symbol{symbol}, tp.httpProxy)
	go group.Subscribe(ch, d)
	return ch
}

// SubscribeAll - subscribing to all quotes with interval
func (tp *TradesProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	bufLength := len(tp.symbols)
	ch := make(chan schemas.ResultChannel, 2*bufLength)

	for _, group := range tp.groups {
		go group.Subscribe(ch, d)
		time.Sleep(100 * time.Millisecond)
	}
	return ch
}
