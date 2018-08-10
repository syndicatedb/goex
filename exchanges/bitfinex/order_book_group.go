package bitfinex

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"unicode"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

//Event - Bitfinex Websocket event
type event struct {
	Event     string `json:"event"`
	Code      int64  `json:"code"`
	Msg       string `json:"msg"`
	Channel   string `json:"channel"`
	ChanID    int64  `json:"chanId"`
	Symbol    string `json:"symbol"`
	Precision string `json:"prec"`
	Frequency string `json:"freq"`
	Length    string `json:"len"`
	Pair      string `json:"pair"`
	Key       string `json:"key"`
}

// OrderBookSubsMessage - message that will be sent to Bitfinex to subscribe
type orderBookSubsMessage struct {
	Event     string `json:"event"`
	Channel   string `json:"channel"`
	Symbol    string `json:"symbol"`
	Precision string `json:"prec"`
	Frequency string `json:"freq"`
	Length    string `json:"len"`
}

// OrderBookGroup - order book group structure
type OrderBookGroup struct {
	symbols []schemas.Symbol

	wsClient   *websocket.Client
	httpClient *httpclient.Client
	httpProxy  proxy.Provider
	subs       map[int64]event
	bus        ordersBus

	sync.RWMutex
}

type ordersBus struct {
	dch        chan []byte
	ech        chan error
	outChannel chan schemas.ResultChannel
}

// NewOrderBookGroup - OrderBookGroup constructor
func NewOrderBookGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *OrderBookGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &OrderBookGroup{
		symbols:    symbols,
		httpProxy:  httpProxy,
		httpClient: httpclient.New(proxyClient),
		subs:       make(map[int64]event),
		bus: ordersBus{
			dch: make(chan []byte, 2*len(symbols)),
			ech: make(chan error, 2*len(symbols)),
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
	ob.bus.outChannel = ch

	ob.listen()
	ob.connect()
	ob.subscribe()
}

// restart - calling start with outChannel.
// need for restarting group after error.
func (ob *OrderBookGroup) restart() {
	ob.Start(ob.bus.outChannel)
}

// connect - creating new WS client and establishing connection
func (ob *OrderBookGroup) connect() {
	ob.wsClient = websocket.NewClient(wsURL, ob.httpProxy)
	if err := ob.wsClient.Connect(); err != nil {
		log.Println("Error connecting to bitfinex API: ", err)
		ob.restart()
		return
	}
	ob.wsClient.Listen(ob.bus.dch, ob.bus.ech)
}

// subscribe - subscribing to books by symbols
func (ob *OrderBookGroup) subscribe() {
	for _, s := range ob.symbols {
		message := orderBookSubsMessage{
			Event:     eventSubscribe,
			Channel:   "book",
			Symbol:    "t" + strings.ToUpper(s.OriginalName),
			Precision: "P0",
			Frequency: "F0",
			Length:    "100",
		}

		if err := ob.wsClient.Write(message); err != nil {
			log.Printf("Error subsciring to %v order books", s.Name)
			ob.restart()
			return
		}
	}
	log.Println("Subscription ok")
}

// listen - listening to updates from WS
func (ob *OrderBookGroup) listen() {
	go func() {
		for msg := range ob.bus.dch {
			ob.parseMessage(msg)
		}
	}()
	go func() {
		for err := range ob.bus.ech {
			log.Printf("[SM] Error listen: %+v", err)
			ob.restart()
			return
		}
	}()
}

// publish - publishing data into outChannel
func (ob *OrderBookGroup) publish(data interface{}, dataType string, err error) {
	ob.bus.outChannel <- schemas.ResultChannel{
		DataType: dataType,
		Data:     data,
		Error:    err,
	}
}

// parseMessage - parsing incoming WS message.
// Calls handleEvent() or handleMessage().
func (ob *OrderBookGroup) parseMessage(msg []byte) {
	t := bytes.TrimLeftFunc(msg, unicode.IsSpace)
	var err error
	// either a channel data array or an event object, raw json encoding
	if bytes.HasPrefix(t, []byte("[")) {
		ob.handleMessage(msg)
	} else if bytes.HasPrefix(t, []byte("{")) {
		if err = ob.handleEvent(msg); err != nil {
			log.Println("Error handling event: ", err)
		}
	} else {
		err = fmt.Errorf("unexpected message: %s", msg)
	}
	if err != nil {
		fmt.Println("[ERROR] handleMessage: ", err, string(msg))
	}
}

// handleEvent - handling event message from WS
func (ob *OrderBookGroup) handleEvent(msg []byte) (err error) {
	var event event
	if err = json.Unmarshal(msg, &event); err != nil {
		return
	}
	if event.Event == eventInfo {
		if event.Code == wsCodeStopping {
			ob.restart()
			return
		}
	}
	if event.Event == eventSubscribed {
		if event.Channel == channelCandles {
			event.Symbol = strings.Replace(event.Key, "trade:1m:t", "", 1)
			event.Pair = event.Symbol
		}
		ob.add(event)
		return
	}
	log.Println("Unprocessed event: ", string(msg))
	return
}

// handleMessage - handling data message from WS
func (ob *OrderBookGroup) handleMessage(msg []byte) {
	var resp []interface{}
	var e event
	var err error

	if err := json.Unmarshal(msg, &resp); err != nil {
		return
	}
	chanID := int64Value(resp[0])
	if chanID > 0 {
		e, err = ob.get(chanID)
		if err != nil {
			log.Println("Error getting subscriptions: ", chanID, err)
			return
		}
	} else {
		return
	}

	if v, ok := resp[1].(string); ok {
		if v == "hb" {
			return
		}
	}
	if v, ok := resp[1].([]interface{}); ok {
		if _, ok := v[0].([]interface{}); ok {
			// handlung snapshot
			orders, dataType := ob.mapSnapshot(e.Symbol, v)
			go ob.publish(orders, dataType, nil)
			return
		}

		// handlng update
		orders := ob.mapOrderBook(e.Symbol, []interface{}{v})
		go ob.publish(orders, "u", nil)
		return
	}

	log.Println("Unrecognized: ", resp)
	return
}

// handleSnapshot - handling snapshot message
func (ob *OrderBookGroup) mapSnapshot(symbol string, data []interface{}) (orders schemas.OrderBook, datatype string) {
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
				Count:  int(o[1].(float64)),
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

// add - adding channel info with it's ID.
// Need for matching symbol with channel ID.
func (ob *OrderBookGroup) add(e event) {
	ob.Lock()
	defer ob.Unlock()
	ob.subs[e.ChanID] = e
}

// get - loading channel info by it's ID
// Need for matching symbol with channel ID.
func (ob *OrderBookGroup) get(chanID int64) (e event, err error) {
	var ok bool
	ob.RLock()
	defer ob.RUnlock()
	if e, ok = ob.subs[chanID]; ok {
		return
	}
	return e, errors.New("subscription not found")
}

// func int64Value(v interface{}) int64 {
// 	if f, ok := v.(float64); ok {
// 		return int64(f)
// 	}
// 	if i, ok := v.(int64); ok {
// 		return i
// 	}
// 	return 0
// }
