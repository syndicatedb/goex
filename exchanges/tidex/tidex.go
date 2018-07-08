package tidex

import (
	"github.com/syndicatedb/goproxy/proxy"

	"github.com/syndicatedb/goex/schemas"
)

const (
	// URL - API endpoint
	apiSymbols = " https://api.tidex.com/api/3/info"
)

var (
	exchangeName = "tidex"
)

/*
Tidex - exchange struct
*/
type Tidex struct {
	credentials       schemas.Credentials
	httpProxy         *proxy.Provider
	OrderBookProvider schemas.OrderBookProvider
	SymbolProvider    schemas.SymbolProvider
}

// New - Tidex constructor. APIKey and APISecret is mandatory, but could be empty
func New(apiKey, apiSecret string) *Tidex {
	return &Tidex{
		credentials: schemas.Credentials{
			APIKey:    apiKey,
			APISecret: apiSecret,
		},
	}
}

func (ex *Tidex) InitProviders() {
	ex.SymbolProvider = NewSymbolsProvider(ex.httpProxy)
}

// SetProxyProvider - setting proxy
func (ex *Tidex) SetProxyProvider(httpProxy *proxy.Provider) {
	ex.httpProxy = httpProxy
}

// GetOrderBookProvider - getter
func (ex *Tidex) GetOrderBookProvider() schemas.OrderBookProvider {
	return ex.OrderBookProvider
}

// GetSymbolProvider - getter
func (ex *Tidex) GetSymbolProvider() schemas.SymbolProvider {
	return ex.SymbolProvider
}
