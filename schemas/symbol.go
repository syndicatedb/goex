package schemas

// Symbol - exchange symbol
type Symbol struct {
	Name           string  `json:"name"`
	OriginalName   string  `json:"original"`
	Coin           string  `json:"coin"`
	BaseCoin       string  `json:"baseCoin"`
	Fee            float64 `json:"fee"`
	MinPrice       float64 `json:"minPrice"`
	MaxPrice       float64 `json:"maxPrice"`
	MinAmount      float64 `json:"minAmount"`
	MaxAmount      float64 `json:"maxAmount"`
	PricePrecision int     `json:"pricePrecision"`
}
