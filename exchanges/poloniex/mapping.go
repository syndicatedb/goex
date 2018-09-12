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
	var err error

	price, err = strconv.ParseFloat(uo.Rate, 64)
	if err != nil {
		log.Println("Error mapping order: ", err)
	}
	amount, err = strconv.ParseFloat(uo.Amount, 64)
	if err != nil {
		log.Println("Error mapping order: ", err)
	}

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
		CreatedAt: 1, // poloniex doesn't return open orders timestamp
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
	var err error

	layout := "2006-01-02 15:04:05"
	tms, err := time.Parse(layout, ut.Date)
	if err != nil {
		log.Println("Error parsing time: ", err)
	}

	price, err = strconv.ParseFloat(ut.Rate, 64)
	if err != nil {
		log.Println("Error mapping trade: ", err)
	}
	amount, err = strconv.ParseFloat(ut.Amount, 64)
	if err != nil {
		log.Println("Error mapping trade: ", err)
	}
	fee, err = strconv.ParseFloat(ut.Fee, 64)
	if err != nil {
		log.Println("Error mapping trade: ", err)
	}

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
	Error           string      `json:"error"`
}

// OrderCancel represents response on successfully cancelled order
type OrderCancel struct {
	Success int    `json:"success"`
	Error   string `json:"error"`
}
