package tidex

import (
	"github.com/syndicatedb/goex/clients"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// TradingProvider - provides quotes/ticker
type TradingProvider struct {
	credentials schemas.Credentials
	httpProxy   proxy.Provider
	httpClient  *clients.HTTP
}

// NewTradingProvider - TradingProvider constructor
func NewTradingProvider(credentials schemas.Credentials, httpProxy proxy.Provider) *TradingProvider {
	proxyClient := httpProxy.NewClient(exchangeName)
	return &TradingProvider{
		credentials: credentials,
		httpProxy:   httpProxy,
		httpClient:  clients.NewHTTP(proxyClient),
	}
}

// Create - creating order
func (trading *TradingProvider) Create(order schemas.Order) (result []schemas.Order, err error) {
	return
}

// Cancel - cancelling order
func (trading *TradingProvider) Cancel(order schemas.Order) (result schemas.Order, err error) {
	return
}

// CancelAll - cancelling all orders
func (trading *TradingProvider) CancelAll() (err error) {
	return
}
