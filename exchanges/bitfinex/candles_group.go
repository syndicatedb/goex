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

// CandlesGroup - bitfinex candles group structure
type CandlesGroup struct {
	symbols []schemas.Symbol

	wsClient   *websocket.Client
	httpClient *httpclient.Client
	httpProxy  proxy.Provider
	subs       map[int64]event
	bus        bus

	sync.RWMutex
}

// NewCandlesGroup - bitfinex candles group constructor
func NewCandlesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *CandlesGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &CandlesGroup{
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

// Get - loading candles snapshot by symbol
func (cg *CandlesGroup) Get() (candles [][]schemas.Candle, err error) {
	var b []byte
	var resp interface{}
	var symbol string

	if len(cg.symbols) == 0 {
		err = errors.New("Symbol is empty")
		return
	}
	for _, symb := range cg.symbols {
		url := apiCandles + "/trade:1m:t" + symb.OriginalName + "/hist"

		query := httpclient.Params()
		query.Set("limit", "200")
		if b, err = cg.httpClient.Get(url, query, false); err != nil {
			return
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			return
		}
		if cand, ok := resp.([]interface{}); ok {
			if len(cand) > 0 {
				candles = append(candles, cg.mapSnapshot(symb.Name, cand))
			}
		}
	}

	return
}

// Start - starting updates
func (cg *CandlesGroup) Start(ch chan schemas.ResultChannel) {
	cg.bus.outChannel = ch

	cg.listen()
	cg.connect()
	cg.subscribe()
}

// restart - calling start with outChannel.
// need for restarting group after error.
func (cg *CandlesGroup) restart() {
	if err := cg.wsClient.Exit(); err != nil {
		log.Println("Error destroying connection: ", err)
	}
	cg.Start(cg.bus.outChannel)
}

// connect - creating new WS client and establishing connection
func (cg *CandlesGroup) connect() {
	cg.wsClient = websocket.NewClient(wsURL, cg.httpProxy)
	if err := cg.wsClient.Connect(); err != nil {
		log.Println("Error connecting to bitfinex API: ", err)
		// cg.restart()
		return
	}
	cg.wsClient.Listen(cg.bus.dch, cg.bus.ech)
}

// subscribe - subscribing to candles by symbols
func (cg *CandlesGroup) subscribe() {
	for _, symb := range cg.symbols {
		message := candlesSubsMessage{
			Event:   eventSubscribe,
			Channel: "candles",
			Key:     "trade:1m:t" + strings.ToUpper(symb.OriginalName),
		}

		if err := cg.wsClient.Write(message); err != nil {
			log.Printf("Error subsciring to %v candles", symb.Name)
			// cg.restart()
			return
		}
	}
	log.Println("Subscription ok")
}

// listen - listening to updates from WS
func (cg *CandlesGroup) listen() {
	go func() {
		for msg := range cg.bus.dch {
			cg.parseMessage(msg)
		}
	}()
	go func() {
		for err := range cg.bus.ech {
			log.Printf("Error listen: %+v", err)
			// cg.restart()
			return
		}
	}()
}

// publish - publishing data into outChannel
func (cg *CandlesGroup) publish(data interface{}, dataType string, err error) {
	cg.bus.outChannel <- schemas.ResultChannel{
		DataType: dataType,
		Data:     data,
		Error:    err,
	}
}

// parseMessage - parsing incoming WS message.
// Calls handleEvent() or handleMessage().
func (cg *CandlesGroup) parseMessage(msg []byte) {
	t := bytes.TrimLeftFunc(msg, unicode.IsSpace)
	var err error
	// either a channel data array or an event object, raw json encoding
	if bytes.HasPrefix(t, []byte("[")) {
		cg.handleMessage(msg)
	} else if bytes.HasPrefix(t, []byte("{")) {
		if err = cg.handleEvent(msg); err != nil {
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
func (cg *CandlesGroup) handleEvent(msg []byte) (err error) {
	var event event
	if err = json.Unmarshal(msg, &event); err != nil {
		return
	}
	if event.Event == eventInfo {
		if event.Code == wsCodeStopping {
			// ob.restart()
			return
		}
	}
	if event.Event == eventSubscribed {
		if event.Channel == channelCandles {
			event.Symbol = strings.Replace(event.Key, "trade:1m:t", "", 1)
			event.Pair = event.Symbol
		}
		cg.add(event)
		return
	}
	log.Println("Unprocessed event: ", string(msg))
	return
}

func (cg *CandlesGroup) handleMessage(msg []byte) {
	var resp []interface{}
	var e event
	var err error

	if err = json.Unmarshal(msg, &resp); err != nil {
		return
	}
	chanID := int64Value(resp[0])
	if chanID > 0 {
		e, err := cg.get(chanID)
		if err != nil {
			log.Println("Error getting subscriptions: ", chanID, err)
			return
		}
	} else {
		return
	}

	if data, ok := resp[1].(string); ok {
		if data == "hb" {
			return
		}
	}
	if data, ok := resp[1].([]interface{}); ok {
		if len(data) == 1 {
			candle := cg.mapUpdate(e.Symbol, data)
			go cg.publish(candle, "u", nil)
			return
		}
		if len(data) > 1 {
			candles := cg.mapSnapshot(e.Symbol, data)
			go cg.publish(candles, "s", nil)
			return
		}

		log.Println("Unrecognized ", resp)
		return
	}
}

// mapSnapshot - mapping incoming candles snapshot message into common []Candle model
func (cg *CandlesGroup) mapSnapshot(symbol string, data []interface{}) (candles []schemas.Candle) {
	for _, c := range data {
		if cand, ok := c.([]interface{}); ok {
			candles = append(candles, schemas.Candle{
				Symbol:    symbol,
				Open:      cand[1].(float64),
				Close:     cand[2].(float64),
				High:      cand[3].(float64),
				Low:       cand[4].(float64),
				Volume:    cand[5].(float64),
				Timestamp: int64(cand[0].(float64)),
			})
		}
	}

	return
}

// mapUpdate - mapping incoming candle update message into common Candle model
func (cg *CandlesGroup) mapUpdate(symbol string, data []interface{}) schemas.Candle {
	return schemas.Candle{
		Symbol:    symbol,
		Open:      data[1].(float64),
		Close:     data[2].(float64),
		High:      data[3].(float64),
		Low:       data[4].(float64),
		Volume:    data[5].(float64),
		Timestamp: int64(data[0].(float64)),
	}
}

// add - adding channel info with it's ID.
// Need for matching symbol with channel ID.
func (cg *CandlesGroup) add(e event) {
	cg.Lock()
	defer cg.Unlock()
	cg.subs[e.ChanID] = e
}

// get - loading channel info by it's ID
// Need for matching symbol with channel ID.
func (cg *CandlesGroup) get(chanID int64) (e event, err error) {
	var ok bool
	cg.RLock()
	defer cg.RUnlock()
	if e, ok = cg.subs[chanID]; ok {
		return
	}
	return e, errors.New("subscription not found")
}
