package exchanges

import (
	"github.com/syndicatedb/goex/exchanges/tidex"
	"github.com/syndicatedb/goex/schemas"
)

// Exchange names
const (
	Tidex = "tidex"
)

// Exchange - exchange methods
type Exchange interface {
	GetSymbolProvider() schemas.SymbolProvider
	GetOrdersProvider() schemas.OrdersProvider
	GetQuotesProvider() schemas.QuotesProvider
	GetTradesProvider() schemas.TradesProvider
}

// New - exchange constructor
func New(opts schemas.Options) Exchange {
	if opts.Name == Tidex {
		return tidex.New(opts)
	}
	return nil
}
