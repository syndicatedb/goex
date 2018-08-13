package schemas

// Candle - exchange cangle (timeframe)
type Candle struct {
	Symbol         string  `json:"symbol" sql:"-"`
	Timestamp      int64   `json:"mts"`
	Discretization int     `json:"d"` // discretization time in seconds
	Open           float64 `json:"open"`
	Close          float64 `json:"close"`
	High           float64 `json:"high"`
	Low            float64 `json:"low"`
	Volume         float64 `json:"volume" sql:",notnull"`
}
