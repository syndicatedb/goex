package schemas

// Quote - common quote/ticker model
type Quote struct {
	Symbol          string `json:"symbol"`
	Price           string `json:"price"`
	High            string `json:"high"`
	Low             string `json:"low"`
	DrawdownValue   string `json:"ddValue"`
	DrawdownPercent string `json:"ddPercent"`
	VolumeBase      string `json:"volumeBase"` // price * volumeQuote
	VolumeQuote     string `json:"volumeQuote"`
}
