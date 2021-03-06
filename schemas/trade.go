package schemas

const (
	Buy  = "buy"
	Sell = "sell"
)

// Trade - common trade model
type Trade struct {
	ID        string  `json:"id"`       // 21490692,
	OrderID   string  `json:"order_id"` // 21490692,
	Symbol    string  `json:"symbol"`
	Type      string  `json:"type"`   // "ask",
	Price     float64 `json:"price"`  // 0.0721605,
	Amount    float64 `json:"amount"` // 0.18422595,
	Fee       float64 `json:"fee"`    // 0.18422595,
	Timestamp int64   `json:"ts"`     // 1531088906
}

// FilterOptions - options for loading trades
type FilterOptions struct {
	Since   int64  // Since time
	Before  int64  // before time
	FromID  string // trade ID, from which the display starts
	Symbols []Symbol
	Limit   int
	Skip    int
	Page    int
}

type Paging struct {
	Count   int64
	Pages   int64
	Current int64
	Limit   int
}
