package schemas

// Quote - common quote/ticker model
type Quote struct {
	Symbol    string `json:"symbol"`
	High      string `json:"high"`    // 0.072871,
	Low       string `json:"low"`     // 0.07022422,
	Avg       string `json:"avg"`     // 0.07154761,
	Volume    string `json:"vol"`     // 322.631549546088,
	VolCur    string `json:"vol_cur"` // 4485.38487862,
	LastTrade string `json:"last"`    // 0.07237814,
	Buy       string `json:"buy"`     // 0.07224127,
	Sell      string `json:"sell"`    // 0.07260591,
	Updated   int64  `json:"updated"` // 1531085854
}
