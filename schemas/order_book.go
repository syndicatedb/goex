package schemas

// OrderBook - common order book model
type OrderBook struct {
	Symbol string  `json:"symbol"`
	Buy    []Order `json:"buy"`
	Sell   []Order `json:"sell"`
}

// Order - common order model
type Order struct {
	ID           string  `json:"id"`
	Symbol       string  `json:"s"`
	Type         string  `json:"t"`
	Price        float64 `json:"p"`
	Amount       float64 `json:"a"`
	AmountFilled float64 `json:"af"`
	Count        int     `json:"c"`
	CreatedAt    int64   `json:"c_at"`
	Remove       int     `json:"r"`
	Status       string  `json:"st"`
}
