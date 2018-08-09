package kucoin

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type tradesResponse struct {
	responseHeader
	Data []interface{} `json:"data"`
}

// TradesGroup - trades group structure
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

// Subscribe - starting trades updates
func (tg *TradesGroup) Subscribe(ch chan schemas.ResultChannel, d time.Duration) {
	for {
		trades, err := tg.Get()
		if err != nil {
			ch <- schemas.ResultChannel{
				Data:  trades,
				Error: err,
			}
			continue
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

// Get - getting trades snapshot from exchange
func (tg *TradesGroup) Get() (trades [][]schemas.Trade, err error) {
	var b []byte
	var resp tradesResponse

	for _, symbol := range tg.symbols {
		query := httpclient.Params()
		query.Set("symbol", symbol.OriginalName)
		query.Set("limit", "200")

		if b, err = tg.httpClient.Get(apiTrades, query, false); err != nil {
			return
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			return
		}
		if !resp.Success {
			err = fmt.Errorf("Error getting trades: %v", err)
			return
		}

		trades = append(trades, tg.mapSnapshot(symbol.Name, resp.Data))
	}

	return
}

func (tg *TradesGroup) mapSnapshot(symbol string, data []interface{}) (trades []schemas.Trade) {
	for _, el := range data {
		if tr, ok := el.([]interface{}); ok {
			trades = append(trades, schemas.Trade{
				Symbol:    symbol,
				Type:      tr[1].(string),
				Price:     tr[2].(float64),
				Amount:    tr[3].(float64),
				Timestamp: int64(tr[0].(float64)),
			})
		}
	}

	return
}
