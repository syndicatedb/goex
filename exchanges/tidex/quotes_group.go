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

// QuotesGroup - group of quotes to group requests
type QuotesGroup struct {
	symbols    []schemas.Symbol
	httpClient *clients.HTTP
}

// NewQuotesGroup - OrderBook constructor
func NewQuotesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *QuotesGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &QuotesGroup{
		symbols:    symbols,
		httpClient: clients.NewHTTP(proxyClient),
	}
}

// SubscribeAll - getting all symbols from Exchange
func (q *QuotesGroup) subscribe(ch chan schemas.Result, d time.Duration) {
	for {
		quotes, err := q.Get()
		if err != nil {
			ch <- schemas.Result{
				Data:  quotes,
				Error: err,
			}
		}
		for _, b := range quotes {
			ch <- schemas.Result{
				Data:  b,
				Error: err,
			}
		}
		time.Sleep(d)
	}
}

// Get - getting all quotes from Exchange
func (q *QuotesGroup) Get() (quotes []schemas.Quote, err error) {
	var b []byte
	var symbols []string
	for _, symbol := range q.symbols {
		symbols = append(symbols, symbol.OriginalName)
	}
	if b, err = q.httpClient.Get(apiQuotes+strings.Join(symbols, "-"), clients.Params(), false); err != nil {
		return
	}
	var resp QuoteResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		fmt.Println("string(b)", string(b))
		return
	}
	for sname, d := range resp {
		name, _, _ := parseSymbol(sname)
		quotes = append(quotes, d.Map(name))
	}
	return
}
