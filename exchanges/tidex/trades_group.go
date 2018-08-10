package tidex

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// TradesGroup - group of quotes to group requests
type TradesGroup struct {
	symbols    []schemas.Symbol
	httpClient *httpclient.Client
}

// NewTradesGroup - OrderBook constructor
func NewTradesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *TradesGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &TradesGroup{
		symbols:    symbols,
		httpClient: httpclient.New(proxyClient),
	}
}

// SubscribeAll - getting all symbols from Exchange
func (q *TradesGroup) subscribe(ch chan schemas.ResultChannel, d time.Duration) {
	for {
		trades, err := q.Get()
		if err != nil {
			ch <- schemas.ResultChannel{
				Data:  trades,
				Error: err,
			}
		}
		for _, b := range trades {
			if len(b) > 0 {
				ch <- schemas.ResultChannel{
					DataType: "s",
					Data:     b,
					Error:    err,
				}
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
	if b, err = q.httpClient.Get(apiTrades+strings.Join(symbols, "-"), httpclient.Params(), false); err != nil {
		return
	}
	var resp Response
	if err = json.Unmarshal(b, &resp); err != nil {
		fmt.Println("Response error:", string(b))
		return
	}
	if resp.Error != "" {
		log.Println("Error in Trades response: ", resp.Error)
		return
	}
	var tradesResponse TradesResponse
	if err = json.Unmarshal(b, &tradesResponse); err != nil {
		fmt.Println("string(b)", string(b))
		return
	}
	for sname, d := range tradesResponse {
		name, _, _ := parseSymbol(sname)
		var symbolTrades []schemas.Trade
		for _, t := range d {
			symbolTrades = append(symbolTrades, t.Map(name))
		}
		trades = append(trades, symbolTrades)
	}
	return
}
