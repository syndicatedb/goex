package exchanges

import (
	"github.com/syndicatedb/goex/exchanges/tidex"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// Exchange names
const (
	Tidex = "tidex"
)

// Exchange - exchange methods
type Exchange interface {
	SetProxyProvider(pp *proxy.Provider)
	InitProviders()
	GetSymbolProvider() schemas.SymbolProvider
	GetOrdersProvider() schemas.OrdersProvider
	GetQuotesProvider() schemas.QuotesProvider
	GetTradesProvider() schemas.TradesProvider
}

// New - exchange constructor
func New(exchangeName, apiKey, apiSecret string) Exchange {
	if exchangeName == Tidex {
		return tidex.New(apiKey, apiSecret)
	}
	return nil
}

// NewPublic - constructor decorator to use only public endpoints
func NewPublic(exchangeName string) Exchange {
	return New(exchangeName, "", "")
}
