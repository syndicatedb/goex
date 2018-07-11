package schemas

// Quote - common quote/ticker model
type Quote struct {
	Symbol    string  `json:"symbol"`
	High      float64 `json:"high"`    // 0.072871,
	Low       float64 `json:"low"`     // 0.07022422,
	Avg       float64 `json:"avg"`     // 0.07154761,
	Volume    float64 `json:"vol"`     // 322.631549546088,
	VolCur    float64 `json:"vol_cur"` // 4485.38487862,
	LastTrade float64 `json:"last"`    // 0.07237814,
	Buy       float64 `json:"buy"`     // 0.07224127,
	Sell      float64 `json:"sell"`    // 0.07260591,
	Updated   int64   `json:"updated"` // 1531085854
}
