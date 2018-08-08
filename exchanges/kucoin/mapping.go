package kucoin

import (
	"strconv"

	"github.com/syndicatedb/goex/schemas"
)

type responseHeader struct {
	Success   bool   `json:"success"`
	Code      string `json:"code"`
	Message   string `json:"msg"`
	Timestamp int64  `json:"timestamp"`
}
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

type quote struct {
	CoinType      string  `json:"coinType"`
	Trading       bool    `json:"trading"`
	Symbol        string  `json:"symbol"`
	LastDealPrice float64 `json:"lastDealPrice"`
	Buy           float64 `json:"buy"`
	Sell          float64 `json:"sell"`
	Change        float64 `json:"change"`
	CoinTypePair  string  `json:"coinTypePair"`
	Sort          int64   `json:"sort"`
	FeeRate       float64 `json:"feeRate"`
	VolValue      float64 `json:"volValue"`
	Plus          bool    `json:"plus"`
	High          float64 `json:"high"`
	DateTime      int64   `json:"datetime"`
	Vol           float64 `json:"vol"`
	Low           float64 `json:"low"`
	ChangeRate    float64 `json:"changeRate"`
}

func (q *quote) Map() schemas.Quote {
	name, _, _ := parseSymbol(q.Symbol)

	return schemas.Quote{
		Symbol:          name,
		Price:           strconv.FormatFloat(q.LastDealPrice, 'f', 8, 64),
		High:            strconv.FormatFloat(q.High, 'f', 8, 64),
		Low:             strconv.FormatFloat(q.Low, 'f', 8, 64),
		DrawdownValue:   strconv.FormatFloat(q.Change, 'f', 8, 64),
		DrawdownPercent: strconv.FormatFloat(q.ChangeRate, 'f', 8, 64),
		VolumeBase:      strconv.FormatFloat(q.Vol, 'f', 8, 64),
		VolumeQuote:     strconv.FormatFloat(q.VolValue, 'f', 8, 64),
	}
}
