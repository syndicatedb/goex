package binance

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/syndicatedb/goex/schemas"
)

type UserBalanceResponse struct {
	MakerCommission  int           `json:"makerCommission"`
	TakerCommission  int           `json:"takerCommission"`
	BuyerCommission  int           `json:"buyerCommission"`
	SellerCommission int           `json:"sellerCommission"`
	CanTrade         bool          `json:"canTrade"`
	CanWithdraw      bool          `json:"canWithdraw"`
	CanDeposit       bool          `json:"canDeposit"`
	UpdateTime       int64         `json:"updateTime"`
	Balances         []UserBalance `json:"balances"`
}

type UserBalance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

func (ubr *UserBalanceResponse) Map() schemas.UserInfo {
	balances := make(map[string]schemas.Balance)
	for _, b := range ubr.Balances {
		free, err := strconv.ParseFloat(b.Free, 64)
		if err != nil {
			log.Println("Error parsing free", err)
		}
		locked, err := strconv.ParseFloat(b.Locked, 64)
		if err != nil {
			log.Println("Error parsing locked", err)
		}
		balances[b.Asset] = schemas.Balance{
			Coin:      b.Asset,
			Available: free,
			InOrders:  locked,
			Total:     free + locked,
		}
	}
	return schemas.UserInfo{
		Balances: balances,
	}
}

type UserOrdersResponse struct {
	Orders []activeOrder
}

type activeOrder struct {
	OrderID          int64  `json:"orderId"`
	Symbol           string `json:"symbol"`
	Price            string `json:"price"`
	OriginalQuantity string `json:"origQty"`
	ExecQuantity     string `json:"executedQty"`
	IcebergQuantity  string `json:"icebergQty"`
	Status           string `json:"status"`
	TimeInForce      string `json:"timeInForce"`
	OrderType        string `json:"type"`
	Side             string `json:"side"`
	StopPrice        string `json:"stopPrice"`
	Time             int64  `json:"time"`
	IsWorking        bool   `json:"isWorking"`
}

func (uor *UserOrdersResponse) Map() (orders []schemas.Order) {
	for _, o := range uor.Orders {
		price, err := strconv.ParseFloat(o.Price, 64)
		if err != nil {
			log.Println("Error mapping price in active orders. Binance:", err)
		}
		amount, err := strconv.ParseFloat(o.OriginalQuantity, 64)
		if err != nil {
			log.Println("Error mapping price in active orders. Binance:", err)
		}
		amountFilled, err := strconv.ParseFloat(o.ExecQuantity, 64)
		if err != nil {
			log.Println("Error mapping price in active orders. Binance:", err)
		}

		orders = append(orders, schemas.Order{
			ID:           strconv.FormatInt(o.OrderID, 10),
			Symbol:       o.Symbol,
			Type:         o.Side,
			Price:        price,
			Amount:       amount,
			AmountFilled: amountFilled,
			Count:        1,
			Remove:       0,
			CreatedAt:    o.Time,
		})
	}
	return
}

type UserTradesResponse struct {
	Trades []UserTrade
}

type UserTrade struct {
	ID              int64  `json:"id"`
	OrderID         int64  `json:"orderId"`
	Price           string `json:"price"`
	Symbol          string `json:"symbol"`
	Quantity        string `json:"qty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
	Time            int64  `json:"time"`
	IsBuyer         bool   `json:"isBuyer"`
	IsMaker         bool   `json:"isMaker"`
	IsBestMatch     bool   `json:"isBestMatch"`
}

func (utr *UserTradesResponse) Map() (trades []schemas.Trade) {
	var side string
	for _, t := range utr.Trades {
		symbol, _, _ := parseSymbol(t.Symbol)

		price, err := strconv.ParseFloat(t.Price, 64)
		if err != nil {
			log.Println("Error mapping price in private trades. Binance:", err)
		}
		amount, err := strconv.ParseFloat(t.Quantity, 64)
		if err != nil {
			log.Println("Error mapping qty in private trades. Binance:", err)
		}
		commission, err := strconv.ParseFloat(t.Commission, 64)
		if err != nil {
			log.Println("Error mapping commission in private trades. Binance:", err)
		}
		if t.IsBuyer {
			side = "BUY"
		} else {
			side = "SELL"
		}
		trades = append(trades, schemas.Trade{
			ID:        fmt.Sprintf("%d", t.ID),
			OrderID:   strconv.FormatInt(t.OrderID, 10),
			Symbol:    symbol,
			Type:      side,
			Price:     price,
			Amount:    amount,
			Fee:       commission,
			Timestamp: t.Time,
		})
	}
	return trades
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

type generalMessage struct {
	EventType string `json:"e"`
	EventTime int64  `json:"E"`
}

type balanceMessage struct {
	generalMessage
	BuyerCommission int       `json:"b"`
	Balances        []balance `json:"B"`
}

// balance - struct for getting balances of user
type balance struct {
	Asset  string `json:"a"`
	Free   string `json:"f"`
	Locked string `json:"l"`
}

func (bm *balanceMessage) Map() schemas.UserInfo {
	balances := make(map[string]schemas.Balance)
	for _, b := range bm.Balances {
		free, err := strconv.ParseFloat(b.Free, 64)
		if err != nil {
			log.Println("Error parsing free", err)
		}
		locked, err := strconv.ParseFloat(b.Locked, 64)
		if err != nil {
			log.Println("Error parsing locked", err)
		}
		balances[b.Asset] = schemas.Balance{
			Coin:      b.Asset,
			Available: free,
			InOrders:  locked,
			Total:     free + locked,
		}
	}
	return schemas.UserInfo{
		Balances: balances,
	}
}

type tradesMessage struct {
	generalMessage
	Symbol               string `json:"s"`
	Side                 string `json:"S"`
	OrderType            string `json:"o"`
	TimeInForce          string `json:"f"`
	Quantity             string `json:"q"`
	OrderPrice           string `json:"p"`
	StopPrice            string `json:"P"`
	CurrentExecutionType string `json:"x"`
	CurrentOrderStatus   string `json:"X"`
	OrderID              int64  `json:"i"`
	TransactionTime      int64  `json:"T"`
	Ignore               int    `json:"O"` // ignore this
	TradeID              int64  `json:"t"`
}

func (tm *tradesMessage) Map() (trades []schemas.Trade) {
	symbol, _, _ := parseSymbol(tm.Symbol)

	price, err := strconv.ParseFloat(tm.OrderPrice, 64)
	if err != nil {
		log.Println("Error mapping price in private trades. Binance:", err)
	}
	amount, err := strconv.ParseFloat(tm.Quantity, 64)
	if err != nil {
		log.Println("Error mapping qty in private trades. Binance:", err)
	}
	trades = append(trades, schemas.Trade{
		ID:        fmt.Sprintf("%d", tm.TradeID),
		OrderID:   strconv.FormatInt(tm.OrderID, 10),
		Symbol:    symbol,
		Type:      strings.ToUpper(tm.Side),
		Price:     price,
		Amount:    amount,
		Timestamp: tm.TransactionTime,
	})
	return trades
}

func (tm *tradesMessage) MapOrder() (orders []schemas.Order) {
	symbol, _, _ := parseSymbol(tm.Symbol)

	price, err := strconv.ParseFloat(tm.OrderPrice, 64)
	if err != nil {
		log.Println("Error mapping price in private trades. Binance:", err)
	}
	amount, err := strconv.ParseFloat(tm.Quantity, 64)
	if err != nil {
		log.Println("Error mapping qty in private trades. Binance:", err)
	}
	orders = append(orders, schemas.Order{
		ID:        strconv.FormatInt(tm.OrderID, 10),
		Symbol:    symbol,
		Type:      strings.ToUpper(tm.Side),
		Price:     price,
		Amount:    amount,
		Count:     1,
		Remove:    0,
		CreatedAt: tm.TransactionTime,
	})
	return orders
}
