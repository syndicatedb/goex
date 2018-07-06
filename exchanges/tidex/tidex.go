package tidex

import "github.com/syndicatedb/goex/schemas"

/*
Tidex - exchange struct
*/
type Tidex struct {
	schemas.Exchange // Extending Exchange
}

// New - Tidex constructor. APIKey and APISecret is mandatory, but could be empty
func New(apiKey, apiSecret string) *Tidex {
	return &Tidex{
		Exchange: schemas.NewExchange(apiKey, apiSecret),
	}
}

// NewPublic - constructor decorator to use only public endpoints
func NewPublic() *Tidex {
	return New("", "")
}

// GetOrderBookProvider - getter
func (ex *Tidex) GetOrderBookProvider() schemas.OrderBookProvider {
	return ex.Exchange.OrderBookProvider
}
