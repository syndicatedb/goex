package bitfinex

import (
	"log"

	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type subsMessage struct {
	Event     string `json:"event"`
	Channel   string `json:"channel"`
	Symbol    string `json:"symbol"`
	Precision string `json:"prec"`
	Frequency string `json:"freq"`
	Length    string `json:"len"`
}

type OrderBookGroup struct {
	symbols   []schemas.Symbol
	wsClient  *websocket.Client
	httpProxy proxy.Provider
	subs      *SubsManager
	bus       ordersBus

	emptySymbols map[string]string
}

type ordersBus struct {
	serviceChannel chan ChannelMessage
	dataChannel    chan schemas.ResultChannel
}

func NewOrderBookGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *OrderBookGroup {
	return &OrderBookGroup{
		symbols:      symbols,
		emptySymbols: make(map[string]string),
		httpProxy:    httpProxy,
		bus: ordersBus{
			serviceChannel: make(chan ChannelMessage),
		},
	}
}

func (ob *OrderBookGroup) start(ch chan schemas.ResultChannel) {
	ob.bus.dataChannel = ch

	ob.listen()
	ob.connect()
	ob.subscribe()
}

func (ob *OrderBookGroup) connect() {
	ws := websocket.NewClient(wsURL, ob.httpProxy)
	if err := ws.Connect(); err != nil {
		log.Println("Error connecting to bitfinex API: ", err)
		return
	}
	// ws.ChangeKeepAlive(true)
	ob.wsClient = ws
}

func (ob *OrderBookGroup) subscribe() {
	var smbls []string
	for _, s := range ob.symbols {
		smbls = append(smbls, s.OriginalName)
	}
	ob.subs = NewSubsManager("books", smbls, ob.wsClient, ob.bus.serviceChannel)
	ob.subs.Subscribe()
}

func (ob *OrderBookGroup) listen() {
	go func() {
		for msg := range ob.bus.serviceChannel {
			orders, datatype := ob.handleMessage(msg)
			if len(orders.Buy) > 0 || len(orders.Sell) > 0 {
				log.Println("DATATYPE", datatype)
				log.Println("ORDERS", orders)
				ob.bus.dataChannel <- schemas.ResultChannel{
					DataType: datatype,
					Data:     orders,
				}
				log.Println("Finished writing to channel in order book group")
			}
		}
	}()
}

func (ob *OrderBookGroup) handleMessage(cm ChannelMessage) (orders schemas.OrderBook, dataType string) {
	data := cm.Data
	symbol := cm.Symbol
	dataType = "update"
	if v, ok := data[1].(string); ok {
		if v == "hb" {
			return
		}
		log.Println("string: ", v)
		return
	}
	if v, ok := data[1].([]interface{}); ok {
		if _, ok := v[0].([]interface{}); ok {
			return ob.handleSnapshot(symbol, v)
		}

		orders = ob.mapOrderBook(symbol, v)
	} else {
		log.Println("Unrecognized: ", data)
	}
	return
}

func (ob *OrderBookGroup) handleSnapshot(symbol string, data []interface{}) (orders schemas.OrderBook, datatype string) {
	orders = ob.mapOrderBook(symbol, data)
	datatype = "snapshot"
	return
}

func (ob *OrderBookGroup) mapOrderBook(symbol string, raw []interface{}) schemas.OrderBook {
	orderBook := schemas.OrderBook{
		Symbol: symbol,
	}
	for i := range raw {
		ordr := ob.mapOrder(symbol, raw[i])
		if ordr.Amount > 0 {
			orderBook.Buy = append(orderBook.Buy, ordr)
		} else {
			orderBook.Sell = append(orderBook.Sell, ordr)
		}
	}

	return orderBook
}

func (ob *OrderBookGroup) mapOrder(symbol string, ordr interface{}) schemas.Order {
	if o, ok := ordr.([]interface{}); ok {
		return schemas.Order{
			Symbol: symbol,
			Price:  o[0].(float64),
			Amount: o[2].(float64),
			Count:  1,
		}
	}
	return schemas.Order{}
}
