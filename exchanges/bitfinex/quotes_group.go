package bitfinex

import (
	"log"

	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type QuotesGroup struct {
	symbols   []schemas.Symbol
	wsClient  *websocket.Client
	httpProxy proxy.Provider
	subs      *SubsManager
	bus       quotesBus
}

type quotesBus struct {
	serviceChannel chan ChannelMessage
	dataChannel    chan schemas.ResultChannel
}

func NewQuotesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *QuotesGroup {
	return &QuotesGroup{
		symbols:   symbols,
		httpProxy: httpProxy,
		bus: quotesBus{
			serviceChannel: make(chan ChannelMessage),
		},
	}
}

// start - starting updates
func (q *QuotesGroup) start(ch chan schemas.ResultChannel) {
	q.bus.dataChannel = ch

	q.listen()
	q.connect()
	q.subscribe()
}

// connect - creating new WS client and establishing connection
func (q *QuotesGroup) connect() {
	ws := websocket.NewClient(wsURL, q.httpProxy)
	if err := ws.Connect(); err != nil {
		log.Println("Error connecting to bitfinex API: ", err)
		return
	}

	q.wsClient = ws
}

// subscribe - subscribing to books by symbols
func (q *QuotesGroup) subscribe() {
	var smbls []string
	for _, s := range q.symbols {
		smbls = append(smbls, s.OriginalName)
	}
	q.subs = NewSubsManager("trades", smbls, q.wsClient, q.bus.serviceChannel)
	q.subs.Subscribe()
}

// listen - listening to updates from WS
func (q *QuotesGroup) listen() {
	go func() {
		for msg := range q.bus.serviceChannel {
			trades, datatype := q.handleMessage(msg)
			if len(trades) > 0 {
				q.bus.dataChannel <- schemas.ResultChannel{
					DataType: datatype,
					Data:     trades,
				}
			}
		}
	}()
}

func (q *QuotesGroup) handleMessage(cm ChannelMessage) (quotes []schemas.Quote, dataType string) {
	return
}
