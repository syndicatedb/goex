package binance

import (
	"errors"
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

// Subscribe - subscribing to quote by one symbol
func (ob *OrdersProvider) Subscribe(symbol schemas.Symbol, d time.Duration) (r chan schemas.ResultChannel) {
	ch := make(chan schemas.ResultChannel)
	group := NewOrderBookGroup([]schemas.Symbol{symbol}, ob.httpProxy)
	go group.Start(ch)
	return ch
}

// SubscribeAll - subscribing all groups
func (ob *OrdersProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	bufLength := len(ob.symbols)
	ch := make(chan schemas.ResultChannel, 2*bufLength)

	for _, orderBook := range ob.books {
		go orderBook.Start(ch)
		time.Sleep(100 * time.Millisecond)
	}
	return ch
}

// Get - getting orderbook snapshot by symbol
func (ob *OrdersProvider) Get(symbol schemas.Symbol) (book schemas.OrderBook, err error) {
	group := NewOrderBookGroup([]schemas.Symbol{symbol}, ob.httpProxy)
	result, err := group.Get()
	if err != nil {
		return schemas.OrderBook{}, err
	}

	if len(result) > 0 {
		return result[0], nil
	}

	return schemas.OrderBook{}, errors.New("Empty orderbook for symbol " + symbol.Name)
}
