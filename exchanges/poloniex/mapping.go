package poloniex

import (
	"log"
	"strconv"
	"time"

	"github.com/syndicatedb/goex/schemas"
)

// UserBalance represents poloniex API user balance response model
type UserBalance struct {
	Available string `json:"available"`
	OnOrders  string `json:"onOrders"`
	BTCValue  string `json:"btcValue"`
}

// Map mapping poloniex user balance data into common balance model
func (ub *UserBalance) Map(coin string) schemas.Balance {
	available, err := strconv.ParseFloat(ub.Available, 64)
	if err != nil {
		log.Println("Error parsing user balance data: ", err)
	}
	onOrders, err := strconv.ParseFloat(ub.OnOrders, 64)
	if err != nil {
		log.Println("Error parsing user balance data: ", err)
	}

	return schemas.Balance{
		Coin:      coin,
		Available: available,
		InOrders:  onOrders,
		Total:     available + onOrders,
	}
}

// UserOrder represents poloniex API user order response model
type UserOrder struct {
	OrderNumber string `json:"orderNumber"`
	Type        string `json:"type"`
	Rate        string `json:"rate"`
	Amount      string `json:"amount"`
	Total       string `json:"total"`
}

// Map mapping incoming order data into commom order model
func (uo *UserOrder) Map(symbol string) schemas.Order {
	var orderType string
	var price, amount float64

	price, _ = strconv.ParseFloat(uo.Rate, 64)
	amount, _ = strconv.ParseFloat(uo.Amount, 64)

	if uo.Type == "sell" {
		orderType = typeSell
	}
	if uo.Type == "buy" {
		orderType = typeBuy
	}

	return schemas.Order{
		ID:        uo.OrderNumber,
		Symbol:    symbol,
		Type:      orderType,
		Price:     price,
		Amount:    amount,
		CreatedAt: 0,
	}
}

// UserTrade represents poloniex API user trade response
type UserTrade struct {
	GlobalTradeID int64  `json:"globalTradeID"`
	TradeID       string `json:"tradeID"`
	Date          string `json:"date"`
	Rate          string `json:"rate"`
	Amount        string `json:"amount"`
	Total         string `json:"total"`
	Fee           string `json:"fee"`
	OrderNumber   string `json:"orderNumber"`
	Type          string `json:"type"`
	Category      string `json:"category"`
}

// Map mapping incoming trades data into common trade model
func (ut *UserTrade) Map(symbol string) schemas.Trade {
	var price, amount, fee float64
	var tradeType string

	layout := "2006-01-02 15:04:05"
	tms, err := time.Parse(layout, ut.Date)
	if err != nil {
		log.Println("Error parsing time: ", err)
	}

	price, _ = strconv.ParseFloat(ut.Rate, 64)
	amount, _ = strconv.ParseFloat(ut.Amount, 64)
	fee, _ = strconv.ParseFloat(ut.Fee, 64)

	if ut.Type == "sell" {
		tradeType = typeSell
	}
	if ut.Type == "buy" {
		tradeType = typeBuy
	}

	return schemas.Trade{
		ID:        strconv.FormatInt(ut.GlobalTradeID, 10),
		OrderID:   ut.OrderNumber,
		Symbol:    symbol,
		Type:      tradeType,
		Price:     price,
		Amount:    amount,
		Fee:       fee,
		Timestamp: tms.Unix() * 1000,
	}
}

// OrderCreate represents response on successfully created order
type OrderCreate struct {
	OrderNumber     string      `json:"orderNumber"`
	ResultingTrades []UserTrade `json:"resultingTrades"`
}

// OrderCancel represents response on successfully cancelled order
type OrderCancel struct {
	Success int `json:"success"`
}
