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

// TradesGroup - group of quotes to group requests
type TradesGroup struct {
	symbols    []schemas.Symbol
	httpClient *clients.HTTP
}

// NewTradesGroup - OrderBook constructor
func NewTradesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *TradesGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &TradesGroup{
		symbols:    symbols,
		httpClient: clients.NewHTTP(proxyClient),
	}
}

// SubscribeAll - getting all symbols from Exchange
func (q *TradesGroup) subscribe(ch chan schemas.Result, d time.Duration) {
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
func (q *TradesGroup) Get() (trades [][]schemas.Trade, err error) {
	var b []byte
	var symbols []string
	for _, symbol := range q.symbols {
		symbols = append(symbols, symbol.OriginalName)
	}
	if b, err = q.httpClient.Get(apiTrades+strings.Join(symbols, "-"), clients.Params(), false); err != nil {
		return
	}
	var resp TradesResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		fmt.Println("string(b)", string(b))
		return
	}
	for sname, d := range resp {
		name, _, _ := parseSymbol(sname)
		var symbolTrades []schemas.Trade
		for _, t := range d {
			symbolTrades = append(symbolTrades, t.Map(name))
		}
		trades = append(trades, symbolTrades)
	}
	return
}
