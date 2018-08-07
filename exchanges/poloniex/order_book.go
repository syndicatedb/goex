package poloniex

import (
	"log"
	"sync"
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type OrdersProvider struct {
	httpProxy proxy.Provider
	symbols   []schemas.Symbol
	groups    []*OrderBookGroup

	sync.Mutex
}

func NewOrdersProvider(httpProxy proxy.Provider) *OrdersProvider {
	return &OrdersProvider{
		httpProxy: httpProxy,
	}
}

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

// Subscribe - subscribing to quote by one symbol
func (ob *OrdersProvider) Subscribe(symbol schemas.Symbol, d time.Duration) (r chan schemas.ResultChannel) {
	ch := make(chan schemas.ResultChannel)
	group := NewOrderBookGroup([]schemas.Symbol{symbol}, ob.httpProxy)
	go group.Start(ch)
	return ch
}

// SubscribeAll - subscribing all groups
func (ob *OrdersProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	log.Println("NEW ORDERBOOK PROVIDER")
	ch := make(chan schemas.ResultChannel)

	for _, gr := range ob.groups {
		go gr.Start(ch)
		time.Sleep(100 * time.Millisecond)
	}

	log.Println("CHANNEL IN ADAPTER IS", ch)
	return ch
}

// Get - getting orderbook snapshot by symbol
func (ob *OrdersProvider) Get(symbol schemas.Symbol) (book schemas.OrderBook, err error) {
	// group := NewOrderBookGroup([]schemas.Symbol{symbol}, ob.httpProxy)
	// return group.Get()
	return
}
