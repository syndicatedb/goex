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
	"xproto/shared/lib"

	"github.com/syndicatedb/goex/internal/websocket"
)

const (
	eventSubscribe  = "subscribe"
	eventSubscribed = "subscribed"
	eventInfo       = "info"

	channelOrderBook = "books"
	channelTrades    = "trades"
	channelCandles   = "candles"
	channelTicker    = "ticker"

	wsCodeStopping = 20051
)

// SubsManager - websocket subscription manager
type SubsManager struct {
	channel     string            // bitfinex ws channel: books, trades
	conn        *websocket.Client // websocket connections
	subs        map[int64]Event   // mapping channel to symbol
	bus         bus
	dataChannel chan ChannelMessage
	symbols     []string
	sync.RWMutex
}

/*
Event - Bitfinex Websocket event
*/
type Event struct {
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

type bus struct {
	data   chan []byte
	errors chan error
}

// TradeSubsMessage - message that will be sent to Bitfinex to subscribe
type TradeSubsMessage struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	Symbol  string `json:"symbol"`
}

// CandleSubsMessage - message that will be sent to Bitfinex to subscribe
type CandleSubsMessage struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	Key     string `json:"key"`
}

// OrderBookSubsMessage - message that will be sent to Bitfinex to subscribe
type OrderBookSubsMessage struct {
	Event     string `json:"event"`
	Channel   string `json:"channel"`
	Symbol    string `json:"symbol"`
	Precision string `json:"prec"`
	Frequency string `json:"freq"`
	Length    string `json:"len"`
}

// ChannelMessage - adding symbol to stream data and passing to handler
type ChannelMessage struct {
	Symbol string
	Data   []interface{}
}

// NewSubsManager - SubsManager constructor
func NewSubsManager(channel string, symbols []string, conn *websocket.Client, dch chan ChannelMessage) *SubsManager {
	log.Println("WS:", conn)
	return &SubsManager{
		channel:     channel,
		dataChannel: dch,
		conn:        conn,
		symbols:     symbols,
		subs:        make(map[int64]Event),
		bus: bus{
			data:   make(chan []byte),
			errors: make(chan error),
		},
	}
}

// Subscribe - subscribing to channel
func (sm *SubsManager) Subscribe() {
	sm.listen()
	sm.start()
	sm.subscribe()
}

// reSubscribe - re-subscribing to channel
// Method is created to gain more control
func (sm *SubsManager) reSubscribe() {
	sm.connect()
	sm.start()
	sm.subscribe()
}

func (sm *SubsManager) connect() {
	if err := sm.conn.Connect(); err != nil {
		log.Println("Error connecting: ", err)
	}
}
func (sm *SubsManager) start() {
	sm.conn.Listen(sm.bus.data, sm.bus.errors)
}

func (sm *SubsManager) listen() {
	go func() {
		for msg := range sm.bus.data {
			sm.parseMessage(msg)
		}
	}()
	go func() {
		for err := range sm.bus.errors {
			log.Printf("[SM] Error listen: %+v", err)
		}
	}()
}

func (sm *SubsManager) parseMessage(msg []byte) {
	// log.Printf("string(msg): %s \n", string(msg))
	t := bytes.TrimLeftFunc(msg, unicode.IsSpace)
	var err error
	// either a channel data array or an event object, raw json encoding
	if bytes.HasPrefix(t, []byte("[")) {
		err = sm.handleChannel(msg)
	} else if bytes.HasPrefix(t, []byte("{")) {
		err = sm.handleEvent(msg)
	} else {
		err = fmt.Errorf("unexpected message: %s", msg)
	}
	if err != nil {
		fmt.Println("[ERROR] handleMessage: ", err, string(msg))
	}
}

func (sm *SubsManager) subscribe() {
	for _, symbol := range sm.symbols {
		sm.subscribeToSymbol(symbol, sm.conn)
	}
}

func (sm *SubsManager) subscribeToSymbol(symbol string, conn *websocket.Client) {
	var message interface{}
	if sm.channel == channelTrades {
		message = TradeSubsMessage{
			Event:   eventSubscribe,
			Channel: channelTrades,
			Symbol:  symbol,
		}
	}
	if sm.channel == channelCandles {
		message = CandleSubsMessage{
			Event:   eventSubscribe,
			Channel: channelCandles,
			Key:     "trade:1m:t" + symbol,
		}
	}
	if sm.channel == channelOrderBook {
		message = OrderBookSubsMessage{
			Event:     eventSubscribe,
			Channel:   "book",
			Symbol:    "t" + strings.ToUpper(symbol),
			Precision: "P0",
			Frequency: "F0",
			Length:    "100",
		}
	}
	if sm.channel == channelTicker {
		message = OrderBookSubsMessage{
			Event:     eventSubscribe,
			Channel:   "ticker",
			Symbol:    "t" + strings.ToUpper(symbol),
			Precision: "P0",
			Frequency: "F0",
			Length:    "100",
		}
	}
	log.Println("Subscribing: ", message)
	if err := conn.Write(message); err != nil {
		fmt.Println("Error subscribing to books: ", err)
	}
}

// Exit - exiting service
func (sm *SubsManager) Exit() error {
	log.Println("Gracefuly exiting")
	if err := sm.conn.Exit(); err != nil {
		log.Println("Error disconnecting ws: ", err)
	}
	return nil
}

func (sm *SubsManager) handleChannel(msg []byte) (err error) {
	var channels []interface{}
	if err = json.Unmarshal(msg, &channels); err != nil {
		return
	}
	chanID := lib.Int64Value(channels[0])
	if chanID > 0 {
		var e Event
		e, err = sm.get(chanID)
		if err != nil {
			log.Println("Error getting subscriptions: ", chanID, err)
			return
		}
		sm.dataChannel <- ChannelMessage{
			Symbol: e.Pair,
			Data:   channels,
		}
	}
	return
}

func (sm *SubsManager) handleEvent(msg []byte) (err error) {
	var event Event
	if err = json.Unmarshal(msg, &event); err != nil {
		return
	}
	if event.Event == eventInfo {
		if event.Code == wsCodeStopping {
			sm.reSubscribe()
			return
		}
	}
	if event.Event == eventSubscribed {
		if event.Channel == channelCandles {
			event.Symbol = strings.Replace(event.Key, "trade:1m:t", "", 1)
			event.Pair = event.Symbol
		}
		sm.add(event)
		return
	}
	log.Println("Unprocessed event: ", string(msg))
	return
}

func (sm *SubsManager) add(e Event) {
	sm.Lock()
	defer sm.Unlock()
	sm.subs[e.ChanID] = e
}

func (sm *SubsManager) get(chanID int64) (e Event, err error) {
	var ok bool
	sm.RLock()
	defer sm.RUnlock()
	if e, ok = sm.subs[chanID]; ok {
		return
	}
	return e, errors.New("subscription not found")
}
