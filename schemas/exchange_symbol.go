package schemas

type ExchangeSymbol struct {
	ID         int     `json:"id"`
	ExchangeID int     `json:"exchangeId"`
	Name       string  `json:"name"`
	Precision  float64 `json:"prec"`
	MinLotSize float64 `json:"minLot"`
	MaxLotSize float64 `json:"maxLot"`
}

type MDSymbol struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
