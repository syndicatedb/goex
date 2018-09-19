package tidex

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/internal/state"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// QuotesGroup - group of quotes to group requests
type QuotesGroup struct {
	symbols    []schemas.Symbol
	httpClient *httpclient.Client
	data       *state.State
}

// NewQuotesGroup - OrderBook constructor
func NewQuotesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *QuotesGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &QuotesGroup{
		symbols:    symbols,
		httpClient: httpclient.New(proxyClient),
		data:       state.New(),
	}
}

// SubscribeAll - getting all symbols from Exchange
func (q *QuotesGroup) subscribe(ch chan schemas.ResultChannel, d time.Duration) {
	for {
		quotes, err := q.Get()
		if err != nil {
			ch <- schemas.ResultChannel{
				Data:     quotes,
				Error:    err,
				DataType: "s",
			}
		}
		for _, b := range quotes {
			ch <- schemas.ResultChannel{
				Data:     b,
				Error:    err,
				DataType: "s",
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
		symbols = append(symbols, symbolToPair(symbol.Name))
	}
	if b, err = q.httpClient.Get(apiQuotes+strings.Join(symbols, "-"), httpclient.Params(), false); err != nil {
		return
	}
	var resp QuoteResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		fmt.Println("[TIDEX] string(b)", string(b))
		return
	}
	for sname, d := range resp {
		name, _, _ := parseSymbol(sname)
		quote := d.Map(name)

		quotes = append(quotes, quote)

	}
	return
}
