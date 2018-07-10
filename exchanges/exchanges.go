package exchanges

import (
	"github.com/syndicatedb/goex/exchanges/tidex"
	"github.com/syndicatedb/goex/schemas"
)

// Exchange names
const (
	Tidex  = "tidex"
	Kucoin = "kucoin"
)

// API - exchange API methods
type API interface {
	SymbolProvider() schemas.SymbolProvider
	OrdersProvider() schemas.OrdersProvider
	QuotesProvider() schemas.QuotesProvider
	TradesProvider() schemas.TradesProvider
	UserProvider() schemas.UserProvider
	TradingProvider() schemas.TradingProvider
}

// New - exchange constructor
func New(opts schemas.Options) API {
	if opts.Name == Tidex {
		return tidex.New(opts)
	}
	return nil
}
