package schemas

// Candle - exchange cangle (timeframe)
type Candle struct {
	ExchangeID     int    `json:"eid"`
	Symbol         string `json:"symbol" sql:"-"`
	SymbolID       int
	Timestamp      int64
	MTS            int64   `json:"mts"`
	Discretization int     `json:"d"`
	Open           float64 `json:"open"`
	Close          float64 `json:"close"`
	High           float64 `json:"high"`
	Low            float64 `json:"low"`
	Volume         float64 `json:"volume" sql:",notnull"`
}
