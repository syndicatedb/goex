package tidex

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/syndicatedb/goex/clients"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// OrderBookGroup - order book
type OrderBookGroup struct {
	symbols    []schemas.Symbol
	httpClient *clients.HTTP
}

// NewOrderBookGroup - OrderBook constructor
func NewOrderBookGroup(symbols []schemas.Symbol, httpProxy *proxy.Provider) *OrderBookGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &OrderBookGroup{
		symbols:    symbols,
		httpClient: clients.NewHTTP(proxyClient),
	}
}

// SubscribeAll - getting all symbols from Exchange
func (ob *OrderBookGroup) subscribe(ch chan schemas.Result, d time.Duration) {
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
func (ob *OrderBookGroup) Get() (book map[string]schemas.OrderBook, err error) {
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
