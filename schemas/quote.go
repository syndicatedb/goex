package schemas

// Quote - common quote/ticker model
type Quote struct {
	Symbol      string  `json:"symbol"`
	Price       float64 `json:"price"`
	High        float64 `json:"high"`
	Low         float64 `json:"low"`
	ChangeValue float64 `json:"changeValue"`
	ChangeRate  float64 `json:"changeRate"`
	VolumeBase  float64 `json:"volumeBase"` // price * volumeQuote
	Volume      float64 `json:"volumeQuote"`
}
