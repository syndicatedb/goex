package bitfinex

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"
	"unicode"

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
	subs       map[int64]event
	bus        bus

	sync.RWMutex
}

// NewOrderBookGroup - OrderBookGroup constructor
func NewOrderBookGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *OrderBookGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &OrderBookGroup{
		symbols:    symbols,
		httpProxy:  httpProxy,
		httpClient: httpclient.New(proxyClient),
		subs:       make(map[int64]event),
		bus: bus{
			dch: make(chan []byte, 2*len(symbols)),
			ech: make(chan error, 2*len(symbols)),
		},
	}
}

// Get - loading order books snapshot by one symbol
func (ob *OrderBookGroup) Get() (books []schemas.OrderBook, err error) {
	var b []byte
	var resp interface{}

	if len(ob.symbols) == 0 {
		err = errors.New("[BITFINEX] Symbol is empty")
		return
	}

	for _, smb := range ob.symbols {
		url := apiOrderBook + "/" + "t" + unparseSymbol(smb.Name) + "/P0"

		if b, err = ob.httpClient.Get(url, httpclient.Params(), false); err != nil {
			return
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			return
		}
		if bks, ok := resp.([]interface{}); ok {
			books = append(books, ob.mapOrderBook("t"+unparseSymbol(smb.Name), bks))
		}

		time.Sleep(2 * time.Second)
	}

	return
}

// Start - starting updates
func (ob *OrderBookGroup) Start(ch chan schemas.ResultChannel) {
	ob.bus.outChannel = ch

	ob.listen()
	ob.connect()
	ob.subscribe()
	ob.collectSnapshots()
}

// restart - calling start with outChannel.
// need for restarting group after error.
func (ob *OrderBookGroup) restart() {
	time.Sleep(5 * time.Second)
	if err := ob.wsClient.Exit(); err != nil {
		log.Println("[BITFINEX] Error destroying connection: ", err)
	}
	ob.Start(ob.bus.outChannel)
}

// connect - creating new WS client and establishing connection
func (ob *OrderBookGroup) connect() {
	ob.wsClient = websocket.NewClient(wsURL, ob.httpProxy)
	if err := ob.wsClient.Connect(); err != nil {
		log.Println("[BITFINEX] Error connecting to bitfinex API: ", err)
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
			Symbol:    "t" + unparseSymbol(s.Name),
			Precision: "P0",
			Frequency: "F0",
			Length:    "100",
		}

		if err := ob.wsClient.Write(message); err != nil {
			log.Printf("[BITFINEX] Error subsciring to %v order books", s.Name)
			ob.restart()
			return
		}
	}
	log.Println("[BITFINEX] Subscription ok")
}

// collectSnapshots getting snapshots by OrderBookGroup symbols and publishing them
func (ob *OrderBookGroup) collectSnapshots() {
	go func() {
		for {
			time.Sleep(snapshotInterval)

			data, err := ob.Get()
			if err != nil {
				ob.publish(nil, "s", err)
			}
			for _, book := range data {
				if len(book.Buy) > 0 || len(book.Sell) > 0 {
					ob.publish(book, "s", nil)
				}
			}
		}
	}()
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
			log.Printf("[BITFINEX] Error listen: %+v", err)
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
			log.Println("[BITFINEX] Error handling event: ", err)
		}
	} else {
		err = fmt.Errorf("[BITFINEX] unexpected message: %s", msg)
	}
	if err != nil {
		fmt.Println("[BITFINEX] Error handleMessage: ", err, string(msg))
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
	log.Println("[BITFINEX] Unprocessed event: ", string(msg))
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
			log.Println("[BITFINEX] Error getting subscriptions: ", chanID, err)
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

	log.Println("[BITFINEX] Unrecognized: ", resp)
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
	// log.Println("SYMBOL", smb)
	smb, _, _ := parseSymbol(symbol)
	orderBook := schemas.OrderBook{
		Symbol: smb,
	}
	for i := range raw {
		if o, ok := raw[i].([]interface{}); ok {
			price := o[0].(float64)
			amount := o[2].(float64)

			ordr := schemas.Order{
				Symbol: smb,
				Price:  price,
				Count:  int(o[1].(float64)),
				Amount: math.Abs(amount),
			}

			if ordr.Count == 0 {
				ordr.Remove = 1
			}

			if amount > 0 {
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
	return e, errors.New("[BITFINEX] subscription not found")
}
