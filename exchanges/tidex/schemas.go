package tidex

// SymbolResponse - symbol response
type SymbolResponse struct {
	ServerTime int64             `json:"server_time"` // 1530993527
	Pairs      map[string]Symbol `json:"pairs"`       //
}

// Symbol - tidex symbol model
type Symbol struct {
	DecimalPlaces float64 `json:"decimal_places"` //  8,
	MinPrice      float64 `json:"min_price"`      //  0.0001,
	MaxPrice      float64 `json:"max_price"`      //  3000,
	MinAmount     float64 `json:"min_amount"`     //  0.001,
	MaxAmount     float64 `json:"max_amount"`     //  10000000,
	MinTotal      float64 `json:"min_total"`      //  1,
	Hidden        float64 `json:"hidden"`         //  0,
	Fee           float64 `json:"fee"`            //  0.1
}
