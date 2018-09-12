package idax

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// SymbolsProvider - order book provider
type SymbolsProvider struct {
	httpClient *httpclient.Client
}

// NewSymbolsProvider - SymbolsProvider constructor
func NewSymbolsProvider(httpProxy proxy.Provider) *SymbolsProvider {
	proxyClient := httpProxy.NewClient(exchangeName)
	return &SymbolsProvider{
		httpClient: httpclient.New(proxyClient),
	}
}

// Get - getting all symbols from Exchange
func (sp *SymbolsProvider) Get() (symbols []schemas.Symbol, err error) {
	var b []byte
	if b, err = sp.httpClient.Get(apiSymbols, httpclient.Params(), false); err != nil {
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
	var data []Symbol
	if err = json.Unmarshal(resp.Data, &data); err != nil {
		return
	}

	for _, d := range data {
		name, coin, baseCoin := parseSymbol(d.PairName)
		symbols = append(symbols, schemas.Symbol{
			Name:     name,
			Coin:     coin,
			BaseCoin: baseCoin,
			Fee:      d.BuyerFeeRate,
			// MinPrice:       d.MinPrice,
			// MaxPrice:       d.MaxPrice,
			MinAmount:      d.MinAmount,
			MaxAmount:      d.MaxAmount,
			PricePrecision: d.PriceDecimalPlace,
			// QuotePrecision: d.QtyDecimalPlace,
		})
	}
	return
}

// Subscribe - getting all symbols from Exchange
func (sp *SymbolsProvider) Subscribe(d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)

	go func() {
		for {
			symbols, err := sp.Get()
			ch <- schemas.ResultChannel{
				Data:  symbols,
				Error: err,
			}
			time.Sleep(d)
		}
	}()
	return ch
}
