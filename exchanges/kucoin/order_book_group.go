package kucoin

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"

	"github.com/syndicatedb/goex/internal/http"
)

type orderbookResponse struct {
	responseHeader
	Data map[string]interface{} `json:"data"`
}

// OrderBookGroup - order book
type OrderBookGroup struct {
	symbols    []schemas.Symbol
	httpClient *httpclient.Client
	// emptySymbols map[string]string
}

// NewOrderBookGroup - OrderBook constructor
func NewOrderBookGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *OrderBookGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &OrderBookGroup{
		symbols:    symbols,
		httpClient: httpclient.New(proxyClient),
		// emptySymbols: make(map[string]string),
	}
}

// Subscribe - starting updates for symbols
func (ob *OrderBookGroup) Subscribe(ch chan schemas.ResultChannel, d time.Duration) {
	for {
		book, err := ob.Get()
		if err != nil {
			ch <- schemas.ResultChannel{
				Data:  book,
				Error: err,
			}
		}
		for _, b := range book {
			ch <- schemas.ResultChannel{
				DataType: "s",
				Data:     b,
				Error:    err,
			}
		}

		time.Sleep(d)
	}
}

// Get - loading order books snapshot by symbols from exhange
func (ob *OrderBookGroup) Get() (books map[string]schemas.OrderBook, err error) {
	books = make(map[string]schemas.OrderBook)
	var b []byte
	var resp orderbookResponse

	for _, symbol := range ob.symbols {
		query := httpclient.Params()
		query.Set("symbol", symbol.OriginalName)
		query.Set("limit", "200")

		if b, err = ob.httpClient.Get(apiOrderBook, query, false); err != nil {
			return
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			return
		}
		if !resp.Success {
			err = fmt.Errorf("Error getting orderbook: %v", resp.Message)
			return
		}

		ordb := ob.mapSnapshot(symbol.Name, resp.Data)
		books[symbol.Name] = ordb
	}

	return
}

func (ob *OrderBookGroup) mapSnapshot(symbol string, data map[string]interface{}) schemas.OrderBook {
	book := schemas.OrderBook{
		Symbol: symbol,
	}
	log.Println("SYMBOL", symbol)

	if _, ok := data["SELL"]; ok {
		if s, ok := data["SELL"].([]interface{}); ok {
			for _, el := range s {
				if sell, ok := el.([]interface{}); ok {
					book.Sell = append(book.Sell, schemas.Order{
						Symbol: symbol,
						Price:  sell[0].(float64),
						Amount: sell[1].(float64),
						Count:  1,
					})
				}
			}
		}
	}
	if _, ok := data["BUY"]; ok {
		if s, ok := data["BUY"].([]interface{}); ok {
			for _, el := range s {
				if buy, ok := el.([]interface{}); ok {
					book.Buy = append(book.Buy, schemas.Order{
						Symbol: symbol,
						Price:  buy[0].(float64),
						Amount: buy[1].(float64),
						Count:  1,
					})
				}
			}
		}
	}

	return book
}
