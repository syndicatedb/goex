package schemas

// OrderBook - common order book model
type OrderBook struct {
	Buy  []Order
	Sell []Order
}

// Order - common order model
type Order struct {
	ID           string
	Symbol       string
	Type         string
	Price        float64
	Amount       float64
	AmountFilled float64
	Count        int
	CreatedAt    int64
	UpdatedAt    int64
}
