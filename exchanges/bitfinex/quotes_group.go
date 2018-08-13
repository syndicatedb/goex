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

// QuotesGroup - quotes group strcutre
type QuotesGroup struct {
	symbols    []schemas.Symbol
	wsClient   *websocket.Client
	httpClient *httpclient.Client
	httpProxy  proxy.Provider
	subs       map[int64]event
	bus        bus

	sync.RWMutex
}

type quotesBus struct {
	dch        chan []byte
	ech        chan error
	outChannel chan schemas.ResultChannel
}

// NewQuotesGroup - QuotesGroup constructor
func NewQuotesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *QuotesGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &QuotesGroup{
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

// Get - getting quote by one symbol
func (q *QuotesGroup) Get() (quote schemas.Quote, err error) {
	var b []byte
	var resp interface{}
	var symbol string

	if len(q.symbols) == 0 {
		err = errors.New("Symbol is empty")
		return
	}
	symbol = q.symbols[0].OriginalName
	url := apiQuotes + "/" + "t" + strings.ToUpper(symbol)

	if b, err = q.httpClient.Get(url, httpclient.Params(), false); err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if qt, ok := resp.([]interface{}); ok {
		return q.mapQuote(symbol, qt), nil
	}

	err = errors.New("Exchange order books data invalid")
	return
}

// Start - starting updates
func (q *QuotesGroup) Start(ch chan schemas.ResultChannel) {
	q.bus.outChannel = ch

	q.listen()
	q.connect()
	q.subscribe()
}

// restart - calling start with outChannel.
// need for restarting group after error.
func (q *QuotesGroup) restart() {
	if err := q.wsClient.Exit(); err != nil {
		log.Println("Error destroying connection: ", err)
	}
	q.Start(q.bus.outChannel)
}

// connect - creating new WS client and establishing connection
func (q *QuotesGroup) connect() {
	q.wsClient = websocket.NewClient(wsURL, q.httpProxy)
	if err := q.wsClient.Connect(); err != nil {
		log.Println("Error connecting to bitfinex API: ", err)
		return
	}
	q.wsClient.Listen(q.bus.dch, q.bus.ech)
}

// subscribe - subscribing to books by symbols
func (q *QuotesGroup) subscribe() {
	for _, s := range q.symbols {
		message := tickerSubsMessage{
			Event:     eventSubscribe,
			Channel:   "ticker",
			Symbol:    "t" + strings.ToUpper(s.OriginalName),
			Precision: "P0",
			Frequency: "F0",
			Length:    "100",
		}

		if err := q.wsClient.Write(message); err != nil {
			log.Printf("Error subsciring to %v quotes", s.Name)
			q.restart()
			return
		}
	}
	log.Println("Subscription ok")
}

// listen - listening to updates from WS
func (q *QuotesGroup) listen() {
	go func() {
		for msg := range q.bus.dch {
			q.parseMessage(msg)
		}
	}()
	go func() {
		for err := range q.bus.ech {
			log.Printf("Error listen: %+v", err)
			q.restart()
			return
		}
	}()
}

// publish - publishing data into outChannel
func (q *QuotesGroup) publish(data interface{}, dataType string, err error) {
	q.bus.outChannel <- schemas.ResultChannel{
		DataType: dataType,
		Data:     data,
		Error:    err,
	}
}

// parseMessage - parsing incoming WS message.
// Calls handleEvent() or handleMessage().
func (q *QuotesGroup) parseMessage(msg []byte) {
	t := bytes.TrimLeftFunc(msg, unicode.IsSpace)
	var err error
	// either a channel data array or an event object, raw json encoding
	if bytes.HasPrefix(t, []byte("[")) {
		q.handleMessage(msg)
	} else if bytes.HasPrefix(t, []byte("{")) {
		if err = q.handleEvent(msg); err != nil {
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
func (q *QuotesGroup) handleEvent(msg []byte) (err error) {
	var event event
	if err = json.Unmarshal(msg, &event); err != nil {
		return
	}
	if event.Event == eventInfo {
		if event.Code == wsCodeStopping {
			q.restart()
			return
		}
	}
	if event.Event == eventSubscribed {
		if event.Channel == channelCandles {
			event.Symbol = strings.Replace(event.Key, "trade:1m:t", "", 1)
			event.Pair = event.Symbol
		}
		q.add(event)
		return
	}
	log.Println("Unprocessed event: ", string(msg))
	return
}

// handleMessage - handling incoming WS message
func (q *QuotesGroup) handleMessage(msg []byte) {
	var resp []interface{}
	var e event
	var err error

	if err := json.Unmarshal(msg, &resp); err != nil {
		return
	}
	chanID := int64Value(resp[0])
	if chanID > 0 {
		e, err = q.get(chanID)
		if err != nil {
			log.Println("Error getting subscriptions: ", chanID, err)
			return
		}
	} else {
		return
	}

	if ut, ok := resp[1].(string); ok {
		if ut == "hb" {
			return
		}
	}
	if upd, ok := resp[1].([]interface{}); ok {
		dataType := "u"
		quote := q.mapQuote(e.Symbol, upd)
		if len(quote.Symbol) > 0 {
			go q.publish(quote, dataType, nil)
		}
	}

	return
}

// mapQuote - mapping incoming WS message into common Quote model
func (q *QuotesGroup) mapQuote(symbol string, d []interface{}) schemas.Quote {
	smb, _, _ := parseSymbol(symbol)
	volumeBase := d[7].(float64) * d[6].(float64)

	return schemas.Quote{
		Symbol:      smb,
		Price:       d[6].(float64),
		High:        d[8].(float64),
		Low:         d[9].(float64),
		ChangeValue: d[4].(float64),
		ChangeRate:  d[5].(float64),
		VolumeBase:  volumeBase,
		Volume:      d[7].(float64),
	}
}

// add - adding channel info with it's ID.
// Need for matching symbol with channel ID.
func (q *QuotesGroup) add(e event) {
	q.Lock()
	defer q.Unlock()
	q.subs[e.ChanID] = e
}

// get - loading channel info by it's ID
// Need for matching symbol with channel ID.
func (q *QuotesGroup) get(chanID int64) (e event, err error) {
	var ok bool
	q.RLock()
	defer q.RUnlock()
	if e, ok = q.subs[chanID]; ok {
		return
	}
	return e, errors.New("subscription not found")
}
