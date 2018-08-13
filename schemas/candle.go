package schemas

// Candle - common candle model
type Candle struct {
	Symbol    string  `json:"s"`
	Open      float64 `json:"o"`
	Close     float64 `json:"c"`
	High      float64 `json:"h"`
	Low       float64 `json:"l"`
	Volume    float64 `json:"v"`
	Timestamp int64   `json:"mts"`
}
