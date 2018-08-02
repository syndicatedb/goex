package bitfinex

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// OrderBookGroup - order book group structure
type OrderBookGroup struct {
	symbols []schemas.Symbol

	wsClient   *websocket.Client
	httpClient *httpclient.Client
	httpProxy  proxy.Provider
	subs       *SubsManager
	bus        ordersBus
}

type ordersBus struct {
	serviceChannel chan ChannelMessage
	dataChannel    chan schemas.ResultChannel
}

// NewOrderBookGroup - OrderBookGroup constructor
func NewOrderBookGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *OrderBookGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &OrderBookGroup{
		symbols:    symbols,
		httpProxy:  httpProxy,
		httpClient: httpclient.New(proxyClient),
		bus: ordersBus{
			serviceChannel: make(chan ChannelMessage),
		},
	}
}

// Get - loading order books snapshot by one symbol
func (ob *OrderBookGroup) Get() (book schemas.OrderBook, err error) {
	var b []byte
	var resp interface{}
	var symbol string

	if len(ob.symbols) == 0 {
		err = errors.New("Symbol is empty")
		return
	}
	symbol = ob.symbols[0].OriginalName
	url := apiOrderBook + "/" + "t" + strings.ToUpper(symbol) + "/P0"

	if b, err = ob.httpClient.Get(url, httpclient.Params(), false); err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if books, ok := resp.([]interface{}); ok {
		return ob.mapOrderBook(symbol, books), nil
	}

	err = errors.New("Exchange order books data invalid")
	return
}

// Start - starting updates
func (ob *OrderBookGroup) Start(ch chan schemas.ResultChannel) {
	ob.bus.dataChannel = ch

	ob.listen()
	ob.connect()
	ob.subscribe()
}

// connect - creating new WS client and establishing connection
func (ob *OrderBookGroup) connect() {
	ws := websocket.NewClient(wsURL, ob.httpProxy)
	if err := ws.Connect(); err != nil {
		log.Println("Error connecting to bitfinex API: ", err)
		return
	}

	ob.wsClient = ws
}

// subscribe - subscribing to books by symbols
func (ob *OrderBookGroup) subscribe() {
	var smbls []string
	for _, s := range ob.symbols {
		smbls = append(smbls, s.OriginalName)
	}
	ob.subs = NewSubsManager("books", smbls, ob.wsClient, ob.bus.serviceChannel)
	ob.subs.Subscribe()
}

// listen - listening to updates from WS
func (ob *OrderBookGroup) listen() {
	go func() {
		for msg := range ob.bus.serviceChannel {
			orders, datatype := ob.handleMessage(msg)
			log.Println("DATATYPE", datatype)
			log.Printf("Prepared ORDERBOOK MESSAGE %+v", orders)
			if len(orders.Buy) > 0 || len(orders.Sell) > 0 {
				ob.bus.dataChannel <- schemas.ResultChannel{
					DataType: datatype,
					Data:     orders,
				}
			}
		}
	}()
}

// handleMessage - handling message from WS
func (ob *OrderBookGroup) handleMessage(cm ChannelMessage) (orders schemas.OrderBook, dataType string) {
	data := cm.Data
	symbol := cm.Symbol
	dataType = "u"
	if v, ok := data[1].(string); ok {
		if v == "hb" {
			return
		}
		log.Println("string: ", v)
		// return
	}
	if v, ok := data[1].([]interface{}); ok {
		if _, ok := v[0].([]interface{}); ok {
			return ob.handleSnapshot(symbol, v)
		}

		sl := make([]interface{}, 0)
		sl = append(sl, v)
		orders = ob.mapOrderBook(symbol, sl)
	} else {
		log.Println("Unrecognized: ", data)
	}
	return
}

// handleSnapshot - handling snapshot message
func (ob *OrderBookGroup) handleSnapshot(symbol string, data []interface{}) (orders schemas.OrderBook, datatype string) {
	datatype = "s"
	orders = ob.mapOrderBook(symbol, data)
	return
}

// mapOrderBook - mapping incoming books message into commot OrderBook model
func (ob *OrderBookGroup) mapOrderBook(symbol string, raw []interface{}) schemas.OrderBook {
	smb, _, _ := parseSymbol(symbol)
	orderBook := schemas.OrderBook{
		Symbol: smb,
	}
	for i := range raw {
		if o, ok := raw[i].([]interface{}); ok {
			ordr := schemas.Order{
				Symbol: smb,
				Price:  o[0].(float64),
				Count:  int(o[1].(int64)),
				Amount: o[2].(float64),
			}

			if ordr.Count == 0 {
				ordr.Remove = 1
			}

			if ordr.Amount > 0 {
				orderBook.Buy = append(orderBook.Buy, ordr)
			} else {
				orderBook.Sell = append(orderBook.Sell, ordr)
			}
		}
	}

	return orderBook
}
