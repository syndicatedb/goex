package bitfinex

import (
	"log"
	"strconv"

	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// TradesGroup - trades group structure
type TradesGroup struct {
	symbols   []schemas.Symbol
	wsClient  *websocket.Client
	httpProxy proxy.Provider
	subs      *SubsManager
	bus       tradesBus
}

type tradesBus struct {
	serviceChannel chan ChannelMessage
	dataChannel    chan schemas.ResultChannel
}

// NewTradesGroup - TradesGroup constructor
func NewTradesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *TradesGroup {
	return &TradesGroup{
		symbols:   symbols,
		httpProxy: httpProxy,
		bus: tradesBus{
			serviceChannel: make(chan ChannelMessage),
		},
	}
}

// start - starting updates
func (tg *TradesGroup) start(ch chan schemas.ResultChannel) {
	tg.bus.dataChannel = ch

	tg.listen()
	tg.connect()
	tg.subscribe()
}

// connect - creating new WS client and establishing connection
func (tg *TradesGroup) connect() {
	ws := websocket.NewClient(wsURL, tg.httpProxy)
	if err := ws.Connect(); err != nil {
		log.Println("Error connecting to bitfinex API: ", err)
		return
	}

	tg.wsClient = ws
}

// subscribe - subscribing to books by symbols
func (tg *TradesGroup) subscribe() {
	var smbls []string
	for _, s := range tg.symbols {
		smbls = append(smbls, s.OriginalName)
	}
	tg.subs = NewSubsManager("trades", smbls, tg.wsClient, tg.bus.serviceChannel)
	tg.subs.Subscribe()
}

// listen - listening to updates from WS
func (tg *TradesGroup) listen() {
	go func() {
		for msg := range tg.bus.serviceChannel {
			trades, datatype := tg.handleMessage(msg)
			if len(trades) > 0 {
				tg.bus.dataChannel <- schemas.ResultChannel{
					DataType: datatype,
					Data:     trades,
				}
			}
		}
	}()
}

// handleMessage - handling WS message
func (tg *TradesGroup) handleMessage(cm ChannelMessage) (trades [][]schemas.Trade, dataType string) {
	symbol := cm.Symbol
	data := cm.Data

	if ut, ok := data[1].(string); ok {
		if ut == "hb" {
			return
		}
		if ut == "tu" {
			dataType = "update"
			if d, ok := data[2].([]interface{}); ok {
				trades = append(trades, tg.mapTrade(symbol, d))
				return
			}
			log.Printf("Warning: trade update contains no trade info: %+v\n", cm)
			return
		}
	}
	if snap, ok := data[1].([]interface{}); ok {
		dataType = "snapshot"
		for _, trade := range snap {
			if d, ok := trade.([]interface{}); ok {
				trades = append(trades, tg.mapTrade(symbol, d))
			}
		}
	}
	return
}

// mapTrade - mapping incoming WS message into common Trade model
func (tg *TradesGroup) mapTrade(symbol string, d []interface{}) []schemas.Trade {
	trade := schemas.Trade{
		ID:        strconv.FormatFloat(d[0].(float64), 'f', 8, 64),
		Symbol:    symbol,
		Price:     d[3].(float64),
		Amount:    d[2].(float64),
		Timestamp: int64(d[1].(float64)),
	}

	return []schemas.Trade{trade}
}
