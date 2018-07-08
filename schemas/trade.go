package schemas

// Trade - common trade model
type Trade struct {
	ExchangeID int     `json:"eid"`
	ID         string  `json:"id"` // 21490692,
	Symbol     string  `json:"symbol"`
	Type       string  `json:"type"`      // "ask",
	Price      float64 `json:"price"`     // 0.0721605,
	Amount     float64 `json:"amount"`    // 0.18422595,
	Timestamp  int64   `json:"timestamp"` // 1531088906
}
