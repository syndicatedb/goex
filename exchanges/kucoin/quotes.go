package kucoin

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type allQuotesResp struct {
	responseHeader
	Data []quote `json:"data"`
}

type symbolQuoteResp struct {
	responseHeader
	Data quote `json:"data"`
}

// QuotesProvider - quotes provider structure
type QuotesProvider struct {
	symbols    []schemas.Symbol
	httpClient *httpclient.Client
}

// NewQuotesProvider - QuotesProvider constructor
func NewQuotesProvider(httpProxy proxy.Provider) *QuotesProvider {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &QuotesProvider{
		httpClient: httpclient.New(proxyClient),
	}
}

// SetSymbols - getting all symbols from Exchange
func (qp *QuotesProvider) SetSymbols(symbols []schemas.Symbol) schemas.QuotesProvider {
	qp.symbols = symbols
	return qp
}

// Subscribe - subscribing to one symbol ticker updates
func (qp *QuotesProvider) Subscribe(symbol schemas.Symbol, d time.Duration) chan schemas.ResultChannel {
	bufLength := len(qp.symbols)
	ch := make(chan schemas.ResultChannel, 2*bufLength)

	go func() {
		for {
			quote, err := qp.getBySymbol(symbol)
			if err != nil {
				ch <- schemas.ResultChannel{
					Data:     quote,
					Error:    err,
					DataType: "s",
				}
				continue
			}
			ch <- schemas.ResultChannel{
				Data:     quote,
				Error:    err,
				DataType: "s",
			}

			time.Sleep(d)
		}
	}()

	return ch
}

// SubscribeAll - subscribing to all symbols ticker updates
func (qp *QuotesProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	bufLength := len(qp.symbols)
	ch := make(chan schemas.ResultChannel, 2*bufLength)

	go func() {
		for {
			quotes, err := qp.get()
			if err != nil {
				ch <- schemas.ResultChannel{
					Data:     quotes,
					Error:    err,
					DataType: "s",
				}
				continue
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
	}()

	return ch
}

// Get - getting tick by symbol
func (qp *QuotesProvider) Get(symbol schemas.Symbol) (q schemas.Quote, err error) {
	return qp.getBySymbol(symbol)
}

func (qp *QuotesProvider) get() (quotes []schemas.Quote, err error) {
	var b []byte
	var resp allQuotesResp

	if b, err = qp.httpClient.Get(apiTicker, httpclient.Params(), false); err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if !resp.Success {
		err = fmt.Errorf("Error getting ticker: %v", resp.Message)
		return
	}

	for _, qt := range resp.Data {
		quotes = append(quotes, qt.Map())
	}

	return
}

func (qp *QuotesProvider) getBySymbol(symbol schemas.Symbol) (quote schemas.Quote, err error) {
	var b []byte
	var resp symbolQuoteResp

	query := httpclient.Params()
	query.Set("symbol", symbol.Name)
	if b, err = qp.httpClient.Get(apiTicker, query, false); err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if !resp.Success {
		err = fmt.Errorf("Error getting ticker: %v", resp.Message)
		return
	}

	quote = resp.Data.Map()
	return
}

// Unsubscribe closes all connections, unsubscribes from updates
// TODO: unsubscribe method
func (qp *QuotesProvider) Unsubscribe() (err error) {
	log.Println("Unsubsribing...")

	return
}
