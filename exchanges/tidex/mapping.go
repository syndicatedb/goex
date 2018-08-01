package tidex

import (
	"fmt"
	"strings"

	"github.com/syndicatedb/goex/schemas"
)

// Response - common response to get success or fail before parsing
type Response struct {
	Success int    `json:"success"`
	Error   string `json:"error"`
}

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
	Hidden        int     `json:"hidden"`         //  0,
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
	// return schemas.Quote{
	// 	Symbol:    name,
	// 	High:      q.High,
	// 	Low:       q.Low,
	// 	Avg:       q.Avg,
	// 	Volume:    q.Vol,
	// 	VolCur:    q.VolCur,
	// 	LastTrade: q.Last,
	// 	Buy:       q.Buy,
	// 	Sell:      q.Sell,
	// 	Updated:   int64(q.Updated),
	// }
	return schemas.Quote{}
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

// UserInfoResponse - tidex response
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

// Map - mapping Tidex user info to common
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

// UserOrdersResponse - response with user active orders
type UserOrdersResponse struct {
	Success int              `json:"success"`
	Return  map[string]Order `json:"return"`
}

// Order - Tidex user order
type Order struct {
	Pair             string  `json:"pair"`              // "eth_btc",
	Type             string  `json:"type"`              // "sell",
	StartAmount      float64 `json:"start_amount"`      // 13.345,
	Amount           float64 `json:"amount"`            // 12.345,
	Rate             float64 `json:"rate"`              // 485,
	TimestampCreated int64   `json:"timestamp_created"` // 1342448420,
	Status           int     `json:"status"`            // 0
}

// Map - mapping Tidex orders to common
func (uo *UserOrdersResponse) Map() (orders []schemas.Order) {
	for id, o := range uo.Return {
		symbol, _, _ := parseSymbol(o.Pair)
		orders = append(orders,
			schemas.Order{
				ID:           id,
				Type:         strings.ToUpper(o.Type),
				Symbol:       symbol,
				Price:        o.Rate,
				Amount:       o.StartAmount,
				AmountFilled: o.Amount,
				CreatedAt:    o.TimestampCreated,
			},
		)
	}
	return
}

// UserTradesResponse - response with user trades
type UserTradesResponse struct {
	Success int                  `json:"success"`
	Return  map[string]UserTrade `json:"return"`
}

// UserTrade - Tidex trade
type UserTrade struct {
	Pair      string  `json:"pair"`      // "eth_btc",
	Type      string  `json:"type"`      // "ask",
	Amount    float64 `json:"amount"`    // 0.18422595,
	Rate      float64 `json:"rate"`      // 0.0721605,
	OrderID   int64   `json:"order_id"`  // 21490692,
	Timestamp int64   `json:"timestamp"` // 1531088906
}

// Map - mapping Tidex orders to common
func (ut *UserTradesResponse) Map() (trades []schemas.Trade) {
	for id, t := range ut.Return {
		symbol, _, _ := parseSymbol(t.Pair)
		trades = append(trades,
			schemas.Trade{
				ID:        id,
				OrderID:   fmt.Sprintf("%d", t.OrderID),
				Type:      strings.ToUpper(t.Type),
				Symbol:    symbol,
				Price:     t.Rate,
				Amount:    t.Amount,
				Timestamp: t.Timestamp,
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
