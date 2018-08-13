package kucoin

import (
	"fmt"
	"sync"
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// CandlesProvider - kucoin candles provider structure
type CandlesProvider struct {
	httpProxy proxy.Provider
	symbols   []schemas.Symbol
	groups    []*CandlesGroup

	sync.Mutex
}

// NewCandlesProvider - candles provider constructor
func NewCandlesProvider(httpProxy proxy.Provider) *CandlesProvider {
	return &CandlesProvider{
		httpProxy: httpProxy,
	}
}

// SetSymbols - setting symbols and creating groups by symbols chunks
func (cp *CandlesProvider) SetSymbols(symbols []schemas.Symbol) schemas.CandlesProvider {
	slice := make([]schemas.Symbol, len(symbols))
	copy(slice, symbols)
	capacity := orderBookSymbolsLimit
	for {
		if len(slice) <= capacity {
			cp.groups = append(
				cp.groups,
				NewCandlesGroup(slice, cp.httpProxy),
			)
			break
		}
		cp.groups = append(
			cp.groups,
			NewCandlesGroup(slice[0:capacity], cp.httpProxy),
		)

		slice = slice[capacity:]
	}

	return cp
}

// Get - getting candles snapshot by one symbol
func (cp *CandlesProvider) Get(symbol schemas.Symbol) ([]schemas.Candle, error) {
	group := NewCandlesGroup([]schemas.Symbol{symbol}, cp.httpProxy)
	d, err := group.Get()
	if err != nil {
		return nil, err
	}
	if len(d) > 0 {
		return d[0], nil
	}

	return nil, fmt.Errorf("No candles snapshot for %s", symbol.Name)
}

// Subscribe - subscribing to candles data by one symbol
func (cp *CandlesProvider) Subscribe(symbol schemas.Symbol, d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)
	group := NewCandlesGroup([]schemas.Symbol{symbol}, cp.httpProxy)
	go group.Subscribe(ch, d)
	return ch
}

// SubscribeAll - subscribing all groups
func (cp *CandlesProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)

	for _, orderBook := range cp.groups {
		go orderBook.Subscribe(ch, d)
		time.Sleep(100 * time.Millisecond)
	}
	return ch
}
