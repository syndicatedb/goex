package bitfinex

import (
	"sync"
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// OrdersProvider - order book provider structure
type OrdersProvider struct {
	httpProxy proxy.Provider
	symbols   []schemas.Symbol
	books     []*OrderBookGroup

	sync.Mutex
}

// NewOrdersProvider - OrdersProvider constructor
func NewOrdersProvider(httpProxy proxy.Provider) *OrdersProvider {
	return &OrdersProvider{
		httpProxy: httpProxy,
	}
}

// SetSymbols - setting symbols and creating groups by symbols chunks
func (ob *OrdersProvider) SetSymbols(symbols []schemas.Symbol) schemas.OrdersProvider {
	slice := make([]schemas.Symbol, len(symbols))
	copy(slice, symbols)
	capacity := orderBookSymbolsLimit
	for {
		if len(slice) <= capacity {
			ob.books = append(
				ob.books,
				NewOrderBookGroup(slice, ob.httpProxy),
			)
			break
		}
		ob.books = append(
			ob.books,
			NewOrderBookGroup(slice[0:capacity], ob.httpProxy),
		)

		slice = slice[capacity:]
	}
	return ob
}

// GetOrderBook - TODO: rename to 'Get' and add some code
func (ob *OrdersProvider) GetOrderBook(symbol schemas.Symbol) (book schemas.OrderBook, err error) {
	return
}

// Subscribe - getting all symbols from Exchange
func (ob *OrdersProvider) Subscribe(symbol schemas.Symbol, d time.Duration) (r chan schemas.ResultChannel) {
	return
}

// SubscribeAll - subscribing all groups
func (ob *OrdersProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)

	for _, orderBook := range ob.books {
		go orderBook.start(ch)
		time.Sleep(100 * time.Millisecond)
	}
	return ch
}
