package poloniex

import (
	"encoding/json"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

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

// Get - getting symbols data
func (sp *SymbolsProvider) Get() (symbols []schemas.Symbol, err error) {
	var b []byte
	resp := make(map[string]interface{})

	query := httpclient.Params()
	query.Set("command", commandVolumes)
	if b, err = sp.httpClient.Get(restURL, query, false); err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	for k, v := range resp {
		if smb, ok := v.(map[string]interface{}); ok {
			symbols = append(symbols, sp.mapSymbol(k, smb))
		}
	}

	return
}

// mapSymbol - mapping incoming symbol data into common Symbol model
func (sp *SymbolsProvider) mapSymbol(symbol string, data map[string]interface{}) schemas.Symbol {
	name, baseCoin, quoteCoin := parseSymbol(symbol)
	smb := schemas.Symbol{
		Name:           name,
		OriginalName:   symbol,
		BaseCoin:       baseCoin,
		Coin:           quoteCoin,
		PricePrecision: defaultPrecision,
	}

	return smb
}

// Subscribe - getting all symbols from exchange with interval d
func (sp *SymbolsProvider) Subscribe(d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel, 300)

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
