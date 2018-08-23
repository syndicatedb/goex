package kucoin

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
	// Local map to store incremental snapshot for some amount of time
	// Tidex doesn't have updates, only snapshots

	tradesMap := make(map[string]schemas.Trade)

	// Iterator to clean up map from time to time
	i := 0
	for {
		trades, err := tg.Get()
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
						log.Println("Kucoin: Trades updates trades / input / processed: ", len(tradesMap), "/", len(b), "/", len(t))
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
				ID:        tr[5].(string),
				Symbol:    symbol,
				Type:      strings.ToLower(tr[1].(string)),
				Price:     tr[2].(float64),
				Amount:    tr[3].(float64),
				Timestamp: int64(tr[0].(float64)),
			})
		}
	}

	return
}
