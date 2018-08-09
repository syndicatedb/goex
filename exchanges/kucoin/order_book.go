package kucoin

import (
	"fmt"
	"sync"
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// OrdersProvider - order book provider structure
type OrdersProvider struct {
	httpProxy proxy.Provider
	symbols   []schemas.Symbol
	groups    []*OrderBookGroup

	sync.Mutex
}

// NewOrdersProvider - OrdersProvider constructor
func NewOrdersProvider(httpProxy proxy.Provider) *OrdersProvider {
	return &OrdersProvider{
		httpProxy: httpProxy,
	}
}

// SetSymbols - getting all symbols from Exchange
func (ob *OrdersProvider) SetSymbols(symbols []schemas.Symbol) schemas.OrdersProvider {
	ob.symbols = symbols
	slice := make([]schemas.Symbol, len(symbols))
	copy(slice, symbols)
	capacity := orderBookSymbolsLimit
	for {
		if len(slice) <= capacity {
			ob.groups = append(
				ob.groups,
				NewOrderBookGroup(slice, ob.httpProxy),
			)
			break
		}
		ob.groups = append(
			ob.groups,
			NewOrderBookGroup(slice[0:capacity], ob.httpProxy),
		)

		slice = slice[capacity:]
	}
	return ob
}

// Get - getting all symbols from Exchange
func (ob *OrdersProvider) Get(symbol schemas.Symbol) (book schemas.OrderBook, err error) {
	orderBookGroup := NewOrderBookGroup([]schemas.Symbol{symbol}, ob.httpProxy)
	m, err := orderBookGroup.Get()
	if ordr, ok := m[symbol.Name]; ok {
		return ordr, nil
	}

	err = fmt.Errorf("No orderbooks found for %s", symbol.Name)
	return
}

// Subscribe - getting all symbols from Exchange
func (ob *OrdersProvider) Subscribe(symbol schemas.Symbol, d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)
	group := NewOrderBookGroup([]schemas.Symbol{symbol}, ob.httpProxy)
	go group.Subscribe(ch, d)
	return ch
}

// SubscribeAll - getting all symbols from Exchange
func (ob *OrdersProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	bufLength := len(ob.symbols)
	ch := make(chan schemas.ResultChannel, 2*bufLength)

	for _, orderBook := range ob.groups {
		go orderBook.Subscribe(ch, d)
		time.Sleep(100 * time.Millisecond)
	}
	return ch
}
