package schemas

// Quote - common quote/ticker model
type Quote struct {
	Symbol          string  `json:"symbol"`
	Price           float64 `json:"price"`
	High            float64 `json:"high"`
	Low             float64 `json:"low"`
	DrawdownValue   float64 `json:"ddValue"`
	DrawdownPercent float64 `json:"ddPercent"`
	VolumeBase      float64 `json:"volumeBase"` // price * volumeQuote
	VolumeQuote     float64 `json:"volumeQuote"`
}
