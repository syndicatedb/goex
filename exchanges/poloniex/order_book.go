package poloniex

import (
	"log"
	"sync"
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// OrdersProvider - orders provider structure
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

// SetSymbols - setting symbols to order provider and creating groups
func (ob *OrdersProvider) SetSymbols(symbols []schemas.Symbol) schemas.OrdersProvider {
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

// Get - getting orderbook snapshot by symbol
func (ob *OrdersProvider) Get(symbol schemas.Symbol) (book schemas.OrderBook, err error) {
	group := NewOrderBookGroup([]schemas.Symbol{symbol}, ob.httpProxy)
	d, err := group.Get()
	if err != nil {
		return
	}
	if len(d) > 0 {
		book = d[0]
	}

	return
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

	for _, gr := range ob.groups {
		go gr.Start(ch)
		time.Sleep(100 * time.Millisecond)
	}

	log.Println("CHANNEL IN ADAPTER IS", ch)
	return ch
}
