package bitfinex

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/syndicatedb/goex/internal/http"

	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// TradesGroup - trades group structure
type TradesGroup struct {
	symbols []schemas.Symbol

	wsClient   *websocket.Client
	httpClient *httpclient.Client
	httpProxy  proxy.Provider
	subs       map[int64]event
	bus        bus

	sync.RWMutex
}

// NewTradesGroup - TradesGroup constructor
func NewTradesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *TradesGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &TradesGroup{
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

// Get - getting trades snapshot by symbol
func (tg *TradesGroup) Get() (trades []schemas.Trade, err error) {
	if len(tg.symbols) == 0 {
		err = errors.New("No symbols provided")
		return
	}
	for i := range tg.symbols {
		var resp interface{}
		var symbol string
		var b []byte

		symbol = tg.symbols[i].OriginalName
		url := apiTrades + "/" + "t" + strings.ToUpper(symbol) + "/hist"

		if b, err = tg.httpClient.Get(url, httpclient.Params(), false); err != nil {
			return
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			return
		}
		if trds, ok := resp.([]interface{}); ok {
			for _, tr := range trds {
				if t, ok := tr.([]interface{}); ok {
					trades = append(trades, tg.mapTrade(symbol, t))
				}
			}
		}
	}

	return
}

// Start - starting updates
func (tg *TradesGroup) Start(ch chan schemas.ResultChannel) {
	tg.bus.outChannel = ch

	tg.listen()
	tg.connect()
	tg.subscribe()
}

// restart - calling start with outChannel.
// need for restarting group after error.
func (tg *TradesGroup) restart() {
	if err := tg.wsClient.Exit(); err != nil {
		log.Println("Error destroying connection: ", err)
	}
	tg.Start(tg.bus.outChannel)
}

// connect - creating new WS client and establishing connection
func (tg *TradesGroup) connect() {
	tg.wsClient = websocket.NewClient(wsURL, tg.httpProxy)
	if err := tg.wsClient.Connect(); err != nil {
		log.Println("Error connecting to bitfinex API: ", err)
		return
	}
	tg.wsClient.Listen(tg.bus.dch, tg.bus.ech)
}

// subscribe - subscribing to books by symbols
func (tg *TradesGroup) subscribe() {
	for _, s := range tg.symbols {
		message := tradeSubsMessage{
			Event:   eventSubscribe,
			Channel: channelTrades,
			Symbol:  "t" + strings.ToUpper(s.OriginalName),
		}
		if err := tg.wsClient.Write(message); err != nil {
			log.Printf("Error subsciring to %v trades", s.Name)
			tg.restart()
			return
		}
	}
	log.Println("Subscription ok")
}

// listen - listening to updates from WS
func (tg *TradesGroup) listen() {
	go func() {
		for msg := range tg.bus.dch {
			tg.parseMessage(msg)
		}
	}()
	go func() {
		for err := range tg.bus.ech {
			log.Printf("Error listen: %+v", err)
			tg.restart()
			return
		}
	}()
}

// publish - publishing data into outChannel
func (tg *TradesGroup) publish(data interface{}, dataType string, err error) {
	tg.bus.outChannel <- schemas.ResultChannel{
		DataType: dataType,
		Data:     data,
		Error:    err,
	}
}

// parseMessage - parsing incoming WS message.
// Calls handleEvent() or handleMessage().
func (tg *TradesGroup) parseMessage(msg []byte) {
	t := bytes.TrimLeftFunc(msg, unicode.IsSpace)
	var err error
	// either a channel data array or an event object, raw json encoding
	if bytes.HasPrefix(t, []byte("[")) {
		tg.handleMessage(msg)
	} else if bytes.HasPrefix(t, []byte("{")) {
		if err = tg.handleEvent(msg); err != nil {
			log.Println("Error handling event: ", err)
		}
	} else {
		err = fmt.Errorf("unexpected message: %s", msg)
	}
	if err != nil {
		fmt.Println("[ERROR] handleMessage: ", err, string(msg))
	}
}

// handleEvent - handling incoming WS event message
func (tg *TradesGroup) handleEvent(msg []byte) (err error) {
	var event event
	if err = json.Unmarshal(msg, &event); err != nil {
		return
	}
	if event.Event == eventInfo {
		if event.Code == wsCodeStopping {
			tg.restart()
			return
		}
	}
	if event.Event == eventSubscribed {
		if event.Channel == channelCandles {
			event.Symbol = strings.Replace(event.Key, "trade:1m:t", "", 1)
			event.Pair = event.Symbol
		}
		tg.add(event)
		return
	}
	log.Println("Unprocessed event: ", string(msg))
	return
}

// handleMessage - handling incoming WS data message
func (tg *TradesGroup) handleMessage(msg []byte) {
	var resp []interface{}
	var e event
	var err error

	if err := json.Unmarshal(msg, &resp); err != nil {
		return
	}
	chanID := int64Value(resp[0])
	if chanID > 0 {
		e, err = tg.get(chanID)
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
		if ut == "tu" {
			// handling update
			dataType := "u"
			if d, ok := resp[2].([]interface{}); ok {
				trade := tg.mapTrade(e.Symbol, d)
				go tg.publish([]schemas.Trade{trade}, dataType, nil)
				return
			}
		}
	}
	if snap, ok := resp[1].([]interface{}); ok {
		// handling snapshot
		var trades []schemas.Trade
		dataType := "s"
		for _, trade := range snap {
			if d, ok := trade.([]interface{}); ok {
				trades = append(trades, tg.mapTrade(e.Symbol, d))
			}
		}
		go tg.publish(trades, dataType, nil)
		return
	}
	return
}

// mapTrade - mapping incoming WS message into common Trade model
func (tg *TradesGroup) mapTrade(symbol string, d []interface{}) schemas.Trade {
	smb, _, _ := parseSymbol(symbol)
	return schemas.Trade{
		ID:        strconv.FormatFloat(d[0].(float64), 'f', 8, 64),
		Symbol:    smb,
		Price:     d[3].(float64),
		Amount:    d[2].(float64),
		Timestamp: int64(d[1].(float64)),
	}
}

// add - adding channel info with it's ID.
// Need for matching symbol with channel ID.
func (tg *TradesGroup) add(e event) {
	tg.Lock()
	defer tg.Unlock()
	tg.subs[e.ChanID] = e
}

// get - loading channel info by it's ID
// Need for matching symbol with channel ID.
func (tg *TradesGroup) get(chanID int64) (e event, err error) {
	var ok bool
	tg.RLock()
	defer tg.RUnlock()
	if e, ok = tg.subs[chanID]; ok {
		return
	}
	return e, errors.New("subscription not found")
}
