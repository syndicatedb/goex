package tidex

import (
	"github.com/syndicatedb/goproxy/proxy"

	"github.com/syndicatedb/goex/schemas"
)

const (
	// URL - API endpoint
	urlSymbols = " https://api.tidex.com/api/3/info"
)

/*
Tidex - exchange struct
*/
type Tidex struct {
	credentials       schemas.Credentials
	httpProxy         *proxy.Client
	OrderBookProvider schemas.OrderBookProvider
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

// SetProxy - setting proxy
func (ex *Tidex) SetProxy(httpProxy *proxy.Client) {
	ex.httpProxy = httpProxy
}

// GetOrderBookProvider - getter
func (ex *Tidex) GetOrderBookProvider() schemas.OrderBookProvider {
	return ex.OrderBookProvider
}
