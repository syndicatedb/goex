package kucoin

import (
	"fmt"

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

type coin struct {
	WithdrawMinFee    float64 `json:"withdrawMinFee"`
	CoinType          string  `json:"coinType"`
	WithdrawMinAmount float64 `json:"withdrawMinAmount"`
	WithdrawRemark    string  `json:"withdrawRemark"`
	WithdrawFeeRate   float64 `json:"withdrawFeeRate"`
	ConfirmationCount int64   `json:"confirmationCount"`
	Enable            bool    `json:"enable"`
	Name              string  `json:"name"`
	TradePrecision    int64   `json:"tradePrecision"`
	EnableWithdraw    bool    `json:"enableWithdraw"`
	EnableDeposit     bool    `json:"enableDeposit"`
	Coin              string  `json:"coin"`
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
		Symbol:      name,
		Price:       q.LastDealPrice,
		Sell:        q.Sell,
		Buy:         q.Buy,
		High:        q.High,
		Low:         q.Low,
		ChangeValue: q.Change,
		ChangeRate:  q.ChangeRate,
		VolumeBase:  q.Vol,
		Volume:      q.VolValue,
	}
}

/*
UserBalanceResponse http response
{
  "success": true,
  "code": "OK",
  "msg": "Operation succeeded.",
  "timestamp": 1534014768145,
  "data": [
    {
      "coinType": "KCS",
      "balanceStr": "0.0",
      "freezeBalance": 0,
      "balance": 0,
      "freezeBalanceStr": "0.0"
    }
 ]
}
*/
type UserBalanceResponse struct {
	Success   bool          `json:"success"`   // : true,
	Code      string        `json:"code"`      // : "OK",
	Msg       string        `json:"msg"`       // : "Operation succeeded.",
	Timestamp int64         `json:"timestamp"` // : 1534014768145,
	Data      []UserBalance `json:"data"`
}

func (ubr *UserBalanceResponse) Map(prices map[string]float64) schemas.UserInfo {
	balances := make(map[string]schemas.Balance)
	for _, b := range ubr.Data {
		balances[b.CoinType] = schemas.Balance{
			Coin:      b.CoinType,
			Available: b.Balance,
			InOrders:  b.FreezeBalance,
			Total:     b.Balance + b.FreezeBalance,
		}
	}
	return schemas.UserInfo{
		Balances: balances,
		Prices:   prices,
	}
}

/*
UserBalance - kucoin user balance
   {
     "coinType": "KCS",
     "balanceStr": "0.0",
     "freezeBalance": 0,
     "balance": 0,
     "freezeBalanceStr": "0.0"
   }

*/
type UserBalance struct {
	CoinType         string  `json:"coinType"`         // "KCS",
	BalanceStr       string  `json:"balanceStr"`       // "0.0",
	FreezeBalance    float64 `json:"freezeBalance"`    // 0,
	Balance          float64 `json:"balance"`          // 0,
	FreezeBalanceStr string  `json:"freezeBalanceStr"` // "0.0"
}

/*
UserTradesResponse http response
{
  "success": true,
  "code": "OK",
  "msg": "Operation succeeded.",
  "timestamp": 1534017182845,
  "data": data
}
*/
type UserTradesResponse struct {
	Success   bool   `json:"success"`   // : true,
	Code      string `json:"code"`      // : "OK",
	Msg       string `json:"msg"`       // : "Operation succeeded.",
	Timestamp int64  `json:"timestamp"` // : 1534014768145,
	Data      struct {
		Total      int64       `json:"total"`      // 59180,
		FirstPage  bool        `json:"firstPage"`  // true,
		LastPage   bool        `json:"lastPage"`   // false,
		CurrPageNo int64       `json:"currPageNo"` // 1,
		Limit      int         `json:"limit"`      // 12,
		PageNos    int64       `json:"pageNos"`    // 4932
		Datas      []UserTrade `json:"datas"`      // 4932
	} `json:"data"`
}

func (utr *UserTradesResponse) Map() []schemas.Trade {
	var trades []schemas.Trade
	for _, t := range utr.Data.Datas {
		trades = append(trades, schemas.Trade{
			ID:        fmt.Sprintf("%d", t.ID),
			OrderID:   t.OrderID,
			Symbol:    t.CoinType + "-" + t.CoinTypePair,
			Type:      t.DealDirection,
			Price:     t.DealPrice,
			Amount:    t.Amount,
			Fee:       t.Fee,
			Timestamp: t.CreatedAt,
		})
	}
	return trades
}

/*
UserTrade user trade

   {
     "coinType": "CAPP",
     "amount": 2148.25,
     "dealValue": 0.00388833,
     "fee": 2.14825,
     "dealDirection": "BUY",
     "coinTypePair": "BTC",
     "oid": "5b5f0f4025cae61a001c8271",
     "dealPrice": 0.00000181,
     "orderOid": "5b5f0f4025cae61d5840a58d",
     "feeRate": 0.001,
     "createdAt": 1532956480000,
     "id": 1845414,
     "direction": "BUY"
   }
*/
type UserTrade struct {
	CoinType      string  `json:"coinType"`      // "CAPP",
	Amount        float64 `json:"amount"`        // 2148.25,
	DealValue     float64 `json:"dealValue"`     // 0.00388833,
	Fee           float64 `json:"fee"`           // 2.14825,
	DealDirection string  `json:"dealDirection"` // "BUY",
	CoinTypePair  string  `json:"coinTypePair"`  // "BTC",
	Oid           string  `json:"oid"`           // "5b5f0f4025cae61a001c8271",
	DealPrice     float64 `json:"dealPrice"`     // 0.00000181,
	OrderID       string  `json:"orderOid"`      // "5b5f0f4025cae61d5840a58d",
	FeeRate       float64 `json:"feeRate"`       // 0.001,
	CreatedAt     int64   `json:"createdAt"`     // 1532956480000,
	ID            int64   `json:"id"`            // 1845414,
	Direction     string  `json:"direction"`     // "BUY"
}

type UserOrdersResponse struct {
	Success   bool      `json:"success"`   // : true,
	Code      string    `json:"code"`      // : "OK",
	Msg       string    `json:"msg"`       // : "Operation succeeded.",
	Timestamp int64     `json:"timestamp"` // : 1534014768145,
	Data      OrderBook `json:"data"`
}

type OrderBook struct {
	Sell []UserOrder
	Buy  []UserOrder
}

func (ob *OrderBook) Map() (orders []schemas.Order) {
	for _, o := range ob.Buy {
		orders = append(orders, o.Map())
	}
	for _, o := range ob.Sell {
		orders = append(orders, o.Map())
	}
	return
}

type UserOrder struct {
	Oid           string  `json:"oid"`           // "59e59b279bd8d31d093d956e",
	Type          string  `json:"type"`          // "SELL",
	UserOid       string  `json:"userOid"`       // null,
	CoinType      string  `json:"coinType"`      // "KCS",
	CoinTypePair  string  `json:"coinTypePair"`  // "BTC",
	Direction     string  `json:"direction"`     // "SELL",
	Price         float64 `json:"price"`         // 0.1,
	DealAmount    float64 `json:"dealAmount"`    // 0,
	PendingAmount float64 `json:"pendingAmount"` // 100,
	CreatedAt     int64   `json:"createdAt"`     // 1508219688000,
	UpdatedAt     int64   `json:"updatedAt"`     // 1508219688000
}

func (uo *UserOrder) Map() schemas.Order {
	return schemas.Order{
		ID:           uo.Oid,
		Symbol:       uo.CoinType + "-" + uo.CoinTypePair,
		Type:         uo.Direction,
		Price:        uo.Price,
		Amount:       uo.DealAmount,
		AmountFilled: (uo.DealAmount - uo.PendingAmount),
		Count:        1,
		CreatedAt:    uo.CreatedAt,
		Remove:       0,
	}
}

type OrderCreateResponse struct {
	Success   bool   `json:"success"`   // : true,
	Code      string `json:"code"`      // : "OK",
	Msg       string `json:"msg"`       // : "Operation succeeded.",
	Timestamp int64  `json:"timestamp"` // : 1534014768145,
	Data      struct {
		OrderOid string `json:"orderOid"`
	} `json:"data"`
}

type OrderCancelResponse struct {
	Success   bool   `json:"success"`   // : true,
	Code      string `json:"code"`      // : "OK",
	Msg       string `json:"msg"`       // : "Operation succeeded.",
	Timestamp int64  `json:"timestamp"` // : 1534014768145,
	Data      struct {
		OrderOid string `json:"orderOid"`
	} `json:"data"`
}
