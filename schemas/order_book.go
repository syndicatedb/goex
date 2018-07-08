package schemas

// OrderBook - common order book model
type OrderBook struct {
	Buy  []Order
	Sell []Order
}

// Order - common order model
type Order struct {
	ExchangeID int
	Symbol     string
	Type       string
	Price      float64
	Amount     float64
	Count      int
}
