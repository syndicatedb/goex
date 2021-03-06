package idax

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/syndicatedb/goex/schemas"
)

// Response - common response to get success or fail before parsing
type Response struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
}

// Symbol - IDAX symbol model
type Symbol struct {
	PairName          string  `json:"pairName"`          // "UQC_BTC",
	BuyerFeeRate      float64 `json:"buyerFeeRate"`      // 0.002,
	SellerFeeRate     float64 `json:"sellerFeeRate"`     // 0.002,
	MaxAmount         float64 `json:"maxAmount"`         // 100,
	MinAmount         float64 `json:"minAmount"`         // 0.001,
	PriceDecimalPlace int     `json:"priceDecimalPlace"` // 8,
	QtyDecimalPlace   int     `json:"qtyDecimalPlace"`   // 3
}

// Order - IDAX orders
type Order struct {
	OrderSide float64 `json:"orderSide"` //  1,
	Price     float64 `json:"price"`     //  0.0000018,
	Qty       float64 `json:"qty"`       //  1600
}

// Quote - IDAX ticker model
type Quote struct {
	Market            string  `json:"market"`            // "CAPP_BTC",
	BaseCode          string  `json:"baseCode"`          // "CAPP",
	QuoteCode         string  `json:"quoteCode"`         // "BTC",
	LastPrice         float64 `json:"lastPrice"`         // 0.00000183,
	Volume            float64 `json:"volume"`            // 248524.37,
	Total             float64 `json:"total"`             // 0.46334303,
	Change            float64 `json:"change"`            // 0.55,
	High              float64 `json:"high"`              // 0.00000184,
	Low               float64 `json:"low"`               // 0.00000181,
	IsShowIndex       bool    `json:"isShowIndex"`       // true,
	MaxAmount         float64 `json:"maxAmount"`         // 0,
	MinAmount         float64 `json:"minAmount"`         // 0,
	PriceDecimalPlace int     `json:"priceDecimalPlace"` // 8,
	QtyDecimalPlace   int     `json:"qtyDecimalPlace"`   // 2
}

// Map - mapping IDAX model to common model
func (q Quote) Map() schemas.Quote {
	name, _, _ := parseSymbol(q.Market)
	return schemas.Quote{
		Symbol:      name,
		Price:       q.LastPrice,
		High:        q.High,
		Low:         q.Low,
		ChangeValue: q.Change,
		ChangeRate:  q.Change,
		VolumeBase:  q.Volume,
		Volume:      q.Total,
	}
}

// Balance - IDAX balance
type Balance struct {
	CoinID            string  `json:"coinId"`            // "1",
	CoinCode          string  `json:"coinCode"`          // "BTC",
	CoinName          string  `json:"coinName"`          // null,
	Available         float64 `json:"available"`         // 0.02351052,
	Frozen            float64 `json:"frozen"`            // 0,
	SumAmount         float64 `json:"sumAmount"`         // 0.02351052,
	IsDepositEnabled  bool    `json:"isDepositEnabled"`  // true,
	IsWithdrawEnabled bool    `json:"isWithdrawEnabled"` // true,
	Cny               float64 `json:"cny"`               // 0,
	Usd               float64 `json:"usd"`               // 0,
	Btc               float64 `json:"btc"`               // 0,
	CoverImage        string  `json:"coverImage"`        // null,
	Pairs             string  `json:"pairs"`             // null
}

// Map mapping IDAX balance to common
func (b *Balance) Map() schemas.Balance {
	return schemas.Balance{
		Coin:      b.CoinCode,
		Available: b.Available,
		InOrders:  b.Frozen,
		Total:     b.SumAmount,
	}
}

// UserOrder - IDAX Userorders
type UserOrder struct {
	OrderID   string  `json:"orderId"`   //  "2000000000031954030",
	OrderSide int     `json:"orderSide"` //  1,
	PairName  string  `json:"pairName"`  //  "CAPP_USDT",
	Price     float64 `json:"price"`     //  0.006,
	Total     float64 `json:"total"`     //  9000,
	FilledQty float64 `json:"filledQty"` //  0,
	Amount    float64 `json:"amount"`    //  54.00000001,
	Time      string  `json:"time"`      //  "2018-09-02 11:55:38"
}

// Map mapping IDAX order to common
func (uo *UserOrder) Map() schemas.Order {
	symbol, _, _ := parseSymbol(uo.PairName)
	t := time.Now()
	t, _ = time.Parse("2006-01-02 15:04:05", uo.Time)
	createdAt := t.Unix()
	return schemas.Order{
		ID:           uo.OrderID,
		Symbol:       symbol,
		Type:         getOrderTypeBySide(uo.OrderSide),
		Price:        uo.Price,
		Amount:       uo.Amount,
		AmountFilled: uo.FilledQty,
		CreatedAt:    createdAt,
	}
}

// ************************ Below ********************

// TradesResponse - IDAX HTTP response for trades
type TradesResponse map[string][]Trade

// Trade - IDAX trade
type Trade struct {
	Type      string  `json:"type"`      // "ask",
	Price     float64 `json:"price"`     // 0.0721605,
	Amount    float64 `json:"amount"`    // 0.18422595,
	Tid       int64   `json:"tid"`       // 21490692,
	Timestamp int64   `json:"timestamp"` // 1531088906
}

// Map - mapping IDAX trade to common
func (t Trade) Map(symbol string) schemas.Trade {
	var trType string
	if strings.ToLower(t.Type) == "ask" {
		trType = schemas.Buy
	}
	if strings.ToLower(t.Type) == "bid" {
		trType = schemas.Sell
	}
	return schemas.Trade{
		ID:        fmt.Sprintf("%d", t.Tid),
		OrderID:   "",
		Symbol:    symbol,
		Type:      trType,
		Price:     t.Price,
		Amount:    t.Amount,
		Timestamp: t.Timestamp,
	}
}

// UserInfoResponse - IDAX response
type UserInfoResponse struct {
	Return struct {
		Funds map[string]struct {
			Value    float64 `json:"value"`
			InOrders float64 `json:"inOrders"`
		} `json:"funds"`
		Rights struct {
			Info     bool `json:"info"`     // true,
			Trade    bool `json:"trade"`    // true,
			Withdraw bool `json:"withdraw"` // false
		} `json:"rights"`
		TransactionCount int32 `json:"transaction_count"` // 0,
		OpenOrders       int32 `json:"open_orders"`       // 0,
		ServerTime       int64 `json:"server_time"`       // 1531172634
	} `json:"return"`
}

// Map - mapping IDAX user info to common
func (ui *UserInfoResponse) Map() schemas.UserInfo {
	balances := make(map[string]schemas.Balance)
	if len(ui.Return.Funds) > 0 {
		for key, v := range ui.Return.Funds {
			name := strings.ToUpper(key)
			balances[name] = schemas.Balance{
				Coin:      name,
				Total:     (v.Value + v.InOrders),
				InOrders:  v.InOrders,
				Available: v.Value,
			}
		}
	}
	return schemas.UserInfo{
		Access: schemas.Access{
			Read:     ui.Return.Rights.Info,
			Trade:    ui.Return.Rights.Trade,
			Withdraw: ui.Return.Rights.Withdraw,
		},
		Balances:    balances,
		TradesCount: ui.Return.TransactionCount,
		OrdersCount: ui.Return.OpenOrders,
	}
}

// UserTradesResponse - response with user trades
type UserTradesResponse struct {
	Code   int         `json:"code"`
	Msg    string      `json:"msg"`
	Trades []UserTrade `json:"trades"`
}

// UserTrade - IDAX trade
type UserTrade struct {
	Type      string `json:"maker"`     // "ask",
	Quantity  string `json:"quantity"`  // 0.18422595,
	Price     string `json:"price"`     // 0.0721605,
	ID        int64  `json:"id"`        // 21490692,
	Timestamp int64  `json:"timestamp"` // 1531088906
}

// Map - mapping IDAX orders to common
func (ut *UserTradesResponse) Map(symbol string) (trades []schemas.Trade) {
	for _, t := range ut.Trades {
		price, _ := strconv.ParseFloat(t.Price, 64)
		qty, _ := strconv.ParseFloat(t.Quantity, 64)
		trades = append(trades,
			schemas.Trade{
				ID:        strconv.FormatInt(t.ID, 10),
				Type:      strings.ToUpper(t.Type),
				Symbol:    symbol,
				Price:     price,
				Amount:    qty,
				Timestamp: t.Timestamp * 1000,
			},
		)
	}
	return
}

// OrdersCreateResponse - response after order create
type OrdersCreateResponse struct {
	Success int `json:"success"`
	Return  struct {
		Received float64            `json:"received"` // 0.1,
		Remains  float64            `json:"remains"`  // 0,
		OrderID  int64              `json:"order_id"` // 0,
		Funds    map[string]float64 `json:"funds"`    // "eth":325
	} `json:"return"`
}
