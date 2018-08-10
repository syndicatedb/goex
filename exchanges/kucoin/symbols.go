package kucoin

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type symbolsResponse struct {
	responseHeader
	Data []symbol `json:"data"`
}

// SymbolsProvider structure
type SymbolsProvider struct {
	httpClient *httpclient.Client
}

// NewSymbolsProvider  - SymbolsProvider constructor
func NewSymbolsProvider(httpProxy proxy.Provider) *SymbolsProvider {
	proxyClient := httpProxy.NewClient(exchangeName)
	return &SymbolsProvider{
		httpClient: httpclient.New(proxyClient),
	}
}

// Get - loading symbols from exchange
func (sp *SymbolsProvider) Get() (symbols []schemas.Symbol, err error) {
	var b []byte
	var resp symbolsResponse
	if b, err = sp.httpClient.Get(apiSymbols, httpclient.Params(), false); err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if !resp.Success {
		err = fmt.Errorf("Error getting symbols: %v", resp.Message)
		return
	}
	for _, smb := range resp.Data {
		symbols = append(symbols, smb.Map())
	}

	return
}

// Subscribe - subscribing to symbols updates with period 'd'
func (sp *SymbolsProvider) Subscribe(d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)

	go func() {
		for {
			symbols, err := sp.Get()
			ch <- schemas.ResultChannel{
				DataType: "s",
				Data:     symbols,
				Error:    err,
			}

			time.Sleep(d)
		}
	}()

	return ch
}
