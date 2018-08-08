package kucoin

import (
	"github.com/syndicatedb/goex/schemas"
)

type symbol struct {
	CoinType      string  `json:"coinType"`
	Trading       bool    `json:"trading"`
	Symbol        string  `json:"symbol"`
	LastDealPrice float64 `json:"lastDealPrice"`
	Buy           float64 `json:"buy"`
	Sell          float64 `json:"sell"`
	Change        float64 `json:"change"`
	CoinTypePair  string  `json:"coinTypePair"`
	Sort          int     `json:"sort"`
	FeeRate       float64 `json:"feeRate"`
	VolValue      float64 `json:"volValue"`
	High          float64 `json:"high"`
	Datetime      int64   `json:"datetime"`
	Vol           float64 `json:"vol"`
	Low           float64 `json:"low"`
	ChangeRate    float64 `json:"changeRate"`
}

func (s *symbol) Map() schemas.Symbol {
	name, quoteCoin, baseCoin := parseSymbol(s.Symbol)

	return schemas.Symbol{
		Name:         name,
		OriginalName: s.Symbol,
		BaseCoin:     baseCoin,
		Coin:         quoteCoin,
		Fee:          s.FeeRate,
		MinPrice:     s.Low,
		MaxPrice:     s.High,
	}
}
