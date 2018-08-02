package tidex

import (
	"encoding/json"
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
	var resp SymbolResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	for sname, d := range resp.Pairs {
		if d.Hidden == 0 {
			name, coin, baseCoin := parseSymbol(sname)
			symbols = append(symbols, schemas.Symbol{
				Name:         name,
				OriginalName: sname,
				Coin:         coin,
				BaseCoin:     baseCoin,
				Fee:          d.Fee,
				MinPrice:     d.MinPrice,
				MaxPrice:     d.MaxPrice,
				MinAmount:    d.MinAmount,
				MaxAmount:    d.MaxAmount,
			})
		}
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
