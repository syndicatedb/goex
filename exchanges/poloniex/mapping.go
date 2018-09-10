package poloniex

import (
	"log"
	"strconv"

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
