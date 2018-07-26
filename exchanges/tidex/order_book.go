package tidex

import (
	"log"
	"sync"
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// OrdersProvider - order book provider
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

// SetSymbols - getting all symbols from Exchange
func (ob *OrdersProvider) SetSymbols(symbols []schemas.Symbol) schemas.OrdersProvider {
	log.Println("Symbols: ", len(symbols))
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

// GetOrderBook - getting all symbols from Exchange
func (ob *OrdersProvider) GetOrderBook(symbol schemas.Symbol) (book schemas.OrderBook, err error) {
	orderBookGroup := NewOrderBookGroup([]schemas.Symbol{symbol}, ob.httpProxy)
	m, err := orderBookGroup.Get()
	return m[symbol.OriginalName], err
}

// Subscribe - getting all symbols from Exchange
func (ob *OrdersProvider) Subscribe(symbol schemas.Symbol, d time.Duration, ch chan schemas.ResultChannel) {
	// r = make(chan schemas.ResultChannel)
	// return
}

// SubscribeAll - getting all symbols from Exchange
// func (ob *OrdersProvider) SubscribeAll(d time.Duration, ch chan schemas.ResultChannel) {
// 	// ch := make(chan schemas.ResultChannel)

// 	for _, orderBook := range ob.books {
// 		go orderBook.subscribe(ch, d)
// 		time.Sleep(100 * time.Millisecond)
// 	}
// 	// return ch
// }

func (ob *OrdersProvider) SubscribeAll(callback func(string, string, interface{}, error)) {
	// ch := make(chan schemas.ResultChannel)

	for _, orderBook := range ob.books {
		go orderBook.subscribe(callback)
		time.Sleep(100 * time.Millisecond)
	}
	// return ch
}
