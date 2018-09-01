package idax

import (
	"encoding/json"
	"errors"
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
	// Local map to store incremental snapshot for some amount of time
	// IDAX doesn't have updates, only snapshots

	tradesMap := make(map[string]schemas.Trade)

	// Iterator to clean up map from time to time
	i := 0
	for {
		trades, err := q.Get()
		if err != nil {
			ch <- schemas.ResultChannel{
				Data:  trades,
				Error: err,
			}
		} else {
			for _, b := range trades {
				// Cleaning up snapshot map every 300 iterations
				if i > 300 {
					tradesMap = make(map[string]schemas.Trade)
					i = 0
				}
				// If there is trades
				if len(b) > 0 {

					// By default dataType is (s)napshot
					dataType := "s"

					// if trades map is not empty, then we have history
					if len(tradesMap) > 0 {
						dataType = "u" // dataType is (u)pdate
					}

					// For updates or snapshot
					var t []schemas.Trade

					for _, trade := range b {
						if _, ok := tradesMap[trade.ID]; ok == false {
							// Appending to update
							t = append(t, trade)
							// Filling snapshot map
							tradesMap[trade.ID] = trade
						}
					}
					// Sending to listener
					if len(t) > 0 {
						log.Println("IDAX: Trades updates trades / input / processed: ", len(tradesMap), "/", len(b), "/", len(t))
						ch <- schemas.ResultChannel{
							DataType: dataType,
							Data:     b,
							Error:    err,
						}
					}
				}
			}
		}
		i++
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
	if resp.Success != true {
		log.Println("Error in Trades response: ", resp.Message)
		err = errors.New(resp.Message)
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
