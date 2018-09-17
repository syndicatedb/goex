package idax

import (
	"encoding/json"
	"errors"
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
	var quote schemas.Quote
	if len(q.symbols) > 0 {
		for _, symbol := range q.symbols {
			symbols = append(symbols, symbolToPair(symbol.Name))
			if quote, err = q.getQuote(symbol); err != nil {
				return
			}
			quotes = append(quotes, quote)
		}
		return
	}

	if b, err = q.httpClient.Get(getURL(apiQuotes), httpclient.Params(), false); err != nil {
		return
	}
	var resp Response
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if resp.Success != true {
		err = errors.New(resp.Message)
		return
	}
	var data []Quote
	if err = json.Unmarshal(resp.Data, &data); err != nil {
		return
	}
	for _, d := range data {
		quotes = append(quotes, d.Map())
	}
	return
}

// getQuote - getting quote from Exchange by Symbol
func (q *QuotesGroup) getQuote(symbol schemas.Symbol) (quote schemas.Quote, err error) {
	var b []byte
	if b, err = q.httpClient.Get(getURL(apiQuote+"?pairName="+symbolToPair(symbol.Name)), httpclient.Params(), false); err != nil {
		return
	}
	var resp Response
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if resp.Success != true {
		err = errors.New(resp.Message)
		return
	}
	var data Quote
	if err = json.Unmarshal(resp.Data, &data); err != nil {
		return
	}
	return data.Map(), nil
}
