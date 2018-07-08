package schemas

type OrderBook struct {
	Buy  []Order
	Sell []Order
}

type Order struct {
	ExchangeID int
	Symbol     string
	Type       string
	Price      float64
	Amount     float64
	Count      int
}
