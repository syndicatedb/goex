package tidex

import (
	"fmt"

	"github.com/syndicatedb/goex/schemas"
)

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

// OrderBookResponse - Tidex orders book response by symbol
type OrderBookResponse map[string]OrderBook

// OrderBook - Tidex order book
type OrderBook struct {
	Asks [][]float64 `json:"asks"`
	Bids [][]float64 `json:"bids"`
}

// QuoteResponse - Tidex ticker response
type QuoteResponse map[string]Quote

// Quote - Tidex ticker model
type Quote struct {
	High    float64 `json:"high"`    // 0.072871,
	Low     float64 `json:"low"`     // 0.07022422,
	Avg     float64 `json:"avg"`     // 0.07154761,
	Vol     float64 `json:"vol"`     // 322.631549546088,
	VolCur  float64 `json:"vol_cur"` // 4485.38487862,
	Last    float64 `json:"last"`    // 0.07237814,
	Buy     float64 `json:"buy"`     // 0.07224127,
	Sell    float64 `json:"sell"`    // 0.07260591,
	Updated float64 `json:"updated"` // 1531085854
}

// Map - mapping Tidex model to common model
func (q Quote) Map(name string) schemas.Quote {
	return schemas.Quote{
		Name:      name,
		High:      q.High,
		Low:       q.Low,
		Avg:       q.Avg,
		Volume:    q.Vol,
		VolCur:    q.VolCur,
		LastTrade: q.Last,
		Buy:       q.Buy,
		Sell:      q.Sell,
		Updated:   q.Updated,
	}
}

// TradesResponse - Tidex HTTP response for trades
type TradesResponse map[string][]Trade

// Trade - Tidex trade
type Trade struct {
	Type      string  `json:"type"`      // "ask",
	Price     float64 `json:"price"`     // 0.0721605,
	Amount    float64 `json:"amount"`    // 0.18422595,
	Tid       int64   `json:"tid"`       // 21490692,
	Timestamp int64   `json:"timestamp"` // 1531088906
}

// Map - mapping Tidex trade to common
func (t Trade) Map(symbol string) schemas.Trade {
	return schemas.Trade{
		ID:        fmt.Sprintf("%v", t.Tid),
		Symbol:    symbol,
		Type:      t.Type,
		Price:     t.Price,
		Amount:    t.Amount,
		Timestamp: t.Timestamp,
	}
}
