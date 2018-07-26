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

// TradesGroup - group of quotes to group requests
type TradesGroup struct {
	symbols    []schemas.Symbol
	httpClient *clients.HTTP
	httpProxy  proxy.Provider
}

// NewTradesGroup - OrderBook constructor
func NewTradesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *TradesGroup {
	// proxyClient := httpProxy.NewClient(exchangeName)

	return &TradesGroup{
		symbols:   symbols,
		httpProxy: httpProxy,
		// httpClient: clients.NewHTTP(proxyClient),
	}
}

// SubscribeAll - getting all symbols from Exchange
// func (q *TradesGroup) subscribe(ch chan schemas.ResultChannel, d time.Duration) {
// 	for {
// 		trades, err := q.Get()
// 		log.Println("Trades:", len(trades), err)
// 		if err != nil {
// 			ch <- schemas.ResultChannel{
// 				Data:  trades,
// 				Error: err,
// 			}
// 		}
// 		for _, b := range trades {
// 			if len(b) > 0 {
// 				ch <- schemas.ResultChannel{
// 					DataType: "s",
// 					Data:     b,
// 					Error:    err,
// 				}
// 			}
// 		}
// 		time.Sleep(d)
// 	}
// }
func (q *TradesGroup) subscribe(callback func(string, string, interface{}, error)) {
	for {
		log.Println("Before get trades")
		trades, err := q.Get()
		log.Println("After get trades")
		if err != nil {
			for _, b := range trades {
				callback("trades", "", b, err)
			}
		}
		for _, b := range trades {
			callback("trades", "s", b, err)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (t *TradesGroup) SetProxy() {
	proxyClient := t.httpProxy.NewClient(exchangeName)
	t.httpClient = clients.NewHTTP(proxyClient)
}

// Get - getting all quotes from Exchange
func (q *TradesGroup) Get() (trades [][]schemas.Trade, err error) {
	var b []byte
	var symbols []string
	q.SetProxy()
	for _, symbol := range q.symbols {
		symbols = append(symbols, symbol.OriginalName)
	}
	if b, err = q.httpClient.Get(apiTrades+strings.Join(symbols, "-"), clients.Params(), false); err != nil {
		return
	}
	var resp Response
	if err = json.Unmarshal(b, &resp); err != nil {
		log.Println("Response error:", string(b))
		return
	}
	if resp.Error != "" {
		log.Println("Error in Trades response: ", resp.Error)
		return
	}
	var tradesResponse TradesResponse
	if err = json.Unmarshal(b, &tradesResponse); err != nil {
		log.Println("string(b)", string(b))
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
