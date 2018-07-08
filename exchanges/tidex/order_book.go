package tidex

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/syndicatedb/goex/clients"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// OrdersProvider - order book provider
type OrdersProvider struct {
	httpProxy *proxy.Provider
	symbols   []schemas.Symbol
	books     []*OrderBookProvider
	sync.Mutex
}

// OrderBookProvider - order book
type OrderBookProvider struct {
	symbols    []schemas.Symbol
	httpClient *clients.HTTP
}

// NewOrdersProvider - OrdersProvider constructor
func NewOrdersProvider(httpProxy *proxy.Provider) *OrdersProvider {
	return &OrdersProvider{
		httpProxy: httpProxy,
	}
}

// NewOrderBookProvider - OrderBook constructor
func NewOrderBookProvider(symbols []schemas.Symbol, httpProxy *proxy.Provider) *OrderBookProvider {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &OrderBookProvider{
		symbols:    symbols,
		httpClient: clients.NewHTTP(proxyClient),
	}
}

// SetSymbols - getting all symbols from Exchange
func (ob *OrdersProvider) SetSymbols(symbols []schemas.Symbol) schemas.OrdersProvider {
	slice := make([]schemas.Symbol, len(symbols))
	copy(slice, symbols)
	capacity := orderBookSymbolsLimit
	for {
		if len(slice) <= capacity {
			ob.books = append(
				ob.books,
				NewOrderBookProvider(slice, ob.httpProxy),
			)
			break
		}
		ob.books = append(
			ob.books,
			NewOrderBookProvider(slice[0:capacity], ob.httpProxy),
		)

		slice = slice[capacity:]
	}

	return ob
}

// GetOrderBook - getting all symbols from Exchange
func (ob *OrdersProvider) GetOrderBook(symbol schemas.Symbol) (book schemas.OrderBook, err error) {
	orderBookProvider := NewOrderBookProvider([]schemas.Symbol{symbol}, ob.httpProxy)
	m, err := orderBookProvider.Get()
	return m[symbol.OriginalName], err
}

// Subscribe - getting all symbols from Exchange
func (ob *OrdersProvider) Subscribe(symbol schemas.Symbol, d time.Duration) (r chan schemas.Result) {
	return
}

// SubscribeAll - getting all symbols from Exchange
func (ob *OrdersProvider) SubscribeAll(d time.Duration) chan schemas.Result {
	ch := make(chan schemas.Result)

	for _, orderBook := range ob.books {
		go orderBook.subscribe(ch, d)
		time.Sleep(100 * time.Millisecond)
	}
	return ch
}

// SubscribeAll - getting all symbols from Exchange
func (ob *OrderBookProvider) subscribe(ch chan schemas.Result, d time.Duration) {
	for {
		book, err := ob.Get()
		if err != nil {
			ch <- schemas.Result{
				Data:  book,
				Error: err,
			}
		}
		for _, b := range book {
			ch <- schemas.Result{
				Data:  b,
				Error: err,
			}
		}
		time.Sleep(d)
	}
}

// Get - getting all symbols from Exchange
func (ob *OrderBookProvider) Get() (book map[string]schemas.OrderBook, err error) {
	book = make(map[string]schemas.OrderBook)
	var b []byte
	var symbols []string
	for _, symbol := range ob.symbols {
		symbols = append(symbols, symbol.OriginalName)
	}
	if b, err = ob.httpClient.Get(apiOrderBook+strings.Join(symbols, "-"), clients.Params{}); err != nil {
		return
	}
	var resp OrderBookResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		fmt.Println("string(b)", string(b))
		return
	}
	for sname, d := range resp {
		name, _, _ := parseSymbol(sname)
		var b schemas.OrderBook
		for _, o := range d.Asks {
			b.Buy = append(b.Buy, schemas.Order{
				ExchangeID: exchangeID,
				Symbol:     name,
				Price:      o[0],
				Amount:     o[1],
				Count:      1,
			})
		}
		for _, o := range d.Bids {
			b.Sell = append(b.Sell, schemas.Order{
				ExchangeID: exchangeID,
				Symbol:     name,
				Price:      o[0],
				Amount:     o[1],
				Count:      1,
			})
		}
		if len(b.Sell) > 0 || len(b.Sell) > 0 {
			book[sname] = b
		}
	}
	return
}
