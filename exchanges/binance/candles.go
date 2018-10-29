package binance

import (
	"fmt"
	"sync"
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// CandlesProvider - stub binance candles provider
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

// SetSymbols - stub method for binance candles provider
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

// Get - stub method for binance candles provider
func (cp *CandlesProvider) Get(symbol schemas.Symbol) (candles []schemas.Candle, err error) {
	group := NewCandlesGroup([]schemas.Symbol{symbol}, cp.httpProxy)
	d, err := group.Get()
	if err != nil {
		return nil, err
	}
	if len(d) > 0 {
		return d[0], nil
	}

	return nil, fmt.Errorf("Candles snapshot by %s not found", symbol.Name)
}

// Subscribe - stub method for binance candles provider
func (cp *CandlesProvider) Subscribe(symbol schemas.Symbol, d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)
	group := NewCandlesGroup([]schemas.Symbol{symbol}, cp.httpProxy)
	go group.Start(ch)
	return ch
}

// SubscribeAll - stub method for binance candles provider
func (cp *CandlesProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	bufLength := len(cp.symbols)
	ch := make(chan schemas.ResultChannel, 2*bufLength)

	for _, group := range cp.groups {
		go group.Start(ch)
		time.Sleep(100 * time.Millisecond)
	}
	return ch
}

// Unsubscribe closes all connections, unsubscribes from updates
func (cp *CandlesProvider) Unsubscribe() (err error) {
	for _, book := range cp.groups {
		if err := book.Stop(); err != nil {
			return err
		}
	}

	return
}
