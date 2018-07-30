package bitfinex

// Symbol - bitfinex symbol model
type Symbol struct {
	Pair           string `json:"pair"`
	PricePrecision int64  `json:"price_precision"`
	InitialMargin  string `json:"initial_margin"`
	MinMargin      string `json:"minimum_margin"`
	MaxOrderSize   string `json:"maximum_order_size"`
	MinOrderSize   string `json:"minimum_order_size"`
	Expiration     string `json:"expiration"`
}
