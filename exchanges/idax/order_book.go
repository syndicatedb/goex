package idax

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// OrdersProvider - order book provider
type OrdersProvider struct {
	httpClient *httpclient.Client
	symbols    []schemas.Symbol
	sync.Mutex
}

// NewOrdersProvider - OrdersProvider constructor
func NewOrdersProvider(httpProxy proxy.Provider) *OrdersProvider {
	return &OrdersProvider{
		httpClient: httpclient.New(httpProxy.NewClient(exchangeName)),
	}
}

// SetSymbols - getting all symbols from Exchange
func (ob *OrdersProvider) SetSymbols(symbols []schemas.Symbol) schemas.OrdersProvider {
	ob.symbols = symbols
	return ob
}

// Get - getting all symbols from Exchange
func (ob *OrdersProvider) Get(symbol schemas.Symbol) (book schemas.OrderBook, err error) {
	var b []byte

	params := httpclient.Params()
	params.Set("pair", symbolToPair(symbol.Name))
	if b, err = ob.httpClient.Get(apiOrderBook, params, false); err != nil {
		return
	}
	var resp Response
	if err = json.Unmarshal(b, &resp); err != nil {
		fmt.Println("Response error:", string(b))
		return
	}
	if resp.Success != true {
		log.Println("[IDAX] Error in Order response: ", resp.Message)
		err = errors.New(resp.Message)
		return
	}
	var orders []Order
	if err = json.Unmarshal(resp.Data, &orders); err != nil {
		fmt.Println("Order Response error:", string(b))
		return
	}

	for _, o := range orders {
		book.Symbol = symbol.Name
		order := schemas.Order{
			Symbol: symbol.Name,
			Price:  o.Price,
			Amount: o.Qty,
			Count:  1,
		}
		if o.OrderSide == 1 {
			book.Buy = append(book.Buy, order)
		} else {
			book.Sell = append(book.Sell, order)
		}
	}
	return
}

// Subscribe - getting all symbols from Exchange
func (ob *OrdersProvider) Subscribe(symbol schemas.Symbol, d time.Duration) (ch chan schemas.ResultChannel) {
	ch = make(chan schemas.ResultChannel, 100)
	go ob.subscribe(symbol, d, ch)
	return ch
}

// SubscribeAll - getting all symbols from Exchange
func (ob *OrdersProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	bufLength := 2 * len(ob.symbols)
	ch := make(chan schemas.ResultChannel, bufLength)

	for _, symbol := range ob.symbols {
		go ob.subscribe(symbol, d, ch)
		time.Sleep(100 * time.Millisecond)
	}
	return ch
}

// subscribe - getting all symbols from Exchange
func (ob *OrdersProvider) subscribe(symbol schemas.Symbol, d time.Duration, ch chan schemas.ResultChannel) {
	go func() {
		for {
			book, err := ob.Get(symbol)
			ch <- schemas.ResultChannel{
				Data:  book,
				Error: err,
			}
			time.Sleep(d)
		}
	}()
}
