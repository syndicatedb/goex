package tidex

const (
	// URL - API endpoint
	URL = " https://api.tidex.com/api/3/info"
)

// OrderBookProvider - order book provider
type OrderBookProvider struct {
}

// NewOrderBookProvider - OrderBookProvider constructor
func NewOrderBookProvider() *OrderBookProvider {
	return &OrderBookProvider{}
}
