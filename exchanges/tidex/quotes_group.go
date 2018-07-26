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

// QuotesGroup - group of quotes to group requests
type QuotesGroup struct {
	symbols    []schemas.Symbol
	httpClient *clients.HTTP
	httpProxy  proxy.Provider
}

// NewQuotesGroup - OrderBook constructor
func NewQuotesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *QuotesGroup {
	// proxyClient := httpProxy.NewClient(exchangeName)

	return &QuotesGroup{
		symbols:   symbols,
		httpProxy: httpProxy,
		// httpClient: clients.NewHTTP(proxyClient),
	}
}

// SubscribeAll - getting all symbols from Exchange
// func (q *QuotesGroup) subscribe(ch chan schemas.ResultChannel, d time.Duration) {
// 	for {
// 		quotes, err := q.Get()
// 		log.Println("Quotes:", len(quotes), err)
// 		if err != nil {
// 			ch <- schemas.ResultChannel{
// 				Data:  quotes,
// 				Error: err,
// 			}
// 		}
// 		for _, b := range quotes {
// 			ch <- schemas.ResultChannel{
// 				Data:  b,
// 				Error: err,
// 			}
// 		}
// 		time.Sleep(d)
// 	}
// }

func (q *QuotesGroup) subscribe(callback func(string, string, interface{}, error)) {
	for {
		log.Println("Before get quotes")
		quotes, err := q.Get()
		log.Println("After get quotes")
		if err != nil {
			for _, b := range quotes {
				callback("quotes", "", b, err)
			}
		}
		for _, b := range quotes {
			callback("quotes", "s", b, err)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (q *QuotesGroup) SetProxy() {
	proxyClient := q.httpProxy.NewClient(exchangeName)
	q.httpClient = clients.NewHTTP(proxyClient)
}

// Get - getting all quotes from Exchange
func (q *QuotesGroup) Get() (quotes []schemas.Quote, err error) {
	var b []byte
	var symbols []string
	q.SetProxy()
	for _, symbol := range q.symbols {
		symbols = append(symbols, symbol.OriginalName)
	}
	if b, err = q.httpClient.Get(apiQuotes+strings.Join(symbols, "-"), clients.Params(), false); err != nil {
		return
	}
	var resp QuoteResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		log.Println("string(b)", string(b))
		return
	}
	for sname, d := range resp {
		name, _, _ := parseSymbol(sname)
		quotes = append(quotes, d.Map(name))
	}
	return
}
