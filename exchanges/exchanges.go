package exchanges

import (
	"github.com/syndicatedb/goex/exchanges/tidex"
	"github.com/syndicatedb/goex/schemas"
)

// Exchange names
const (
	Tidex = "tidex"
)

type Exchange interface {
	GetOrderBookProvider() schemas.OrderBookProvider
}

func New(exchangeName, apiKey, apiSecret string) Exchange {
	if exchangeName == Tidex {
		return tidex.New(apiKey, apiSecret)
	}
	return nil
}
