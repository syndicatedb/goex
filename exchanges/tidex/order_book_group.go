package tidex

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/syndicatedb/goex/clients"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// OrderBookGroup - order book
type OrderBookGroup struct {
	symbols      []schemas.Symbol
	httpClient   *clients.HTTP
	httpProxy    proxy.Provider
	emptySymbols map[string]string
}

// NewOrderBookGroup - OrderBook constructor
func NewOrderBookGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *OrderBookGroup {
	// proxyClient := httpProxy.NewClient(exchangeName)

	return &OrderBookGroup{
		symbols:   symbols,
		httpProxy: httpProxy,
		// httpClient:   clients.NewHTTP(proxyClient),
		emptySymbols: make(map[string]string),
	}
}

// SubscribeAll - getting all symbols from Exchange
// func (ob *OrderBookGroup) subscribe(ch chan schemas.ResultChannel, d time.Duration) {
// 	i := 0
// 	for {
// 		book, err := ob.Get()
// 		log.Println("Orderbook:", len(book), err)
// 		if err != nil {
// 			ch <- schemas.ResultChannel{
// 				Data:  book,
// 				Error: err,
// 			}
// 		}
// 		for _, b := range book {
// 			ch <- schemas.ResultChannel{
// 				DataType: "s",
// 				Data:     b,
// 				Error:    err,
// 			}
// 		}
// 		i++
// 		if i%5 == 0 {
// 			if len(ob.emptySymbols) > 0 {
// 				log.Println("Empty: ", ob.emptySymbols)
// 			}
// 		}
// 		time.Sleep(d)
// 	}
// }
func (ob *OrderBookGroup) subscribe(callback func(string, string, interface{}, error)) {
	for {
		log.Println("Before get book")
		book, err := ob.Get()
		log.Println("After get book")
		if err != nil {
			for _, b := range book {
				callback("orderbook", "", b, err)
			}
		}
		for _, b := range book {
			callback("orderbook", "s", b, err)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (ob *OrderBookGroup) SetProxy() {
	proxyClient := ob.httpProxy.NewClient(exchangeName)
	ob.httpClient = clients.NewHTTP(proxyClient)
}

// Get - getting all symbols from Exchange
func (ob *OrderBookGroup) Get() (book map[string]schemas.OrderBook, err error) {
	// start := time.Now().UnixNano() / 1000000
	ob.SetProxy()
	book = make(map[string]schemas.OrderBook)
	var by []byte
	var symbols []string
	for _, symbol := range ob.symbols {
		symbols = append(symbols, symbol.OriginalName)
	}
	params := clients.Params()
	params.Set("limit", "2000")
	if by, err = ob.httpClient.Get(apiOrderBook+strings.Join(symbols, "-"), params, false); err != nil {
		return
	}
	// fin := time.Now().UnixNano() / 1000000

	// if (fin - start) > 1000 {
	// 	log.Println("Slow request: ", (fin - start))
	// }
	var resp Response
	if err = json.Unmarshal(by, &resp); err != nil {
		log.Println("Response error:", string(by))
		return
	}
	if resp.Error != "" {
		log.Println("Error in Order response: ", resp.Error)
		return
	}
	var booksResponse OrderBookResponse
	if err = json.Unmarshal(by, &booksResponse); err != nil {
		log.Println("Order Response error:", string(by))
		return
	}
	for sname, d := range booksResponse {
		name, _, _ := parseSymbol(sname)
		var b schemas.OrderBook
		b.Symbol = name
		for _, o := range d.Asks {
			b.Buy = append(b.Buy, schemas.Order{
				Symbol: name,
				Price:  o[0],
				Amount: o[1],
				Count:  1,
			})
		}
		for _, o := range d.Bids {
			b.Sell = append(b.Sell, schemas.Order{
				Symbol: name,
				Price:  o[0],
				Amount: o[1],
				Count:  1,
			})
		}
		if len(b.Sell) > 0 || len(b.Buy) > 0 {
			book[sname] = b
		} else {
			ob.emptySymbols[sname] = sname
		}
	}
	return
}
