package bitfinex

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"

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
	subs       *SubsManager
	bus        quotesBus
}

type quotesBus struct {
	serviceChannel chan ChannelMessage
	dataChannel    chan schemas.ResultChannel
}

// NewQuotesGroup - QuotesGroup constructor
func NewQuotesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *QuotesGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &QuotesGroup{
		symbols:    symbols,
		httpProxy:  httpProxy,
		httpClient: httpclient.New(proxyClient),
		bus: quotesBus{
			serviceChannel: make(chan ChannelMessage),
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
	q.subs = NewSubsManager("ticker", smbls, q.wsClient, q.bus.serviceChannel)
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

// handleMessage - handling incoming WS message
func (q *QuotesGroup) handleMessage(cm ChannelMessage) (quotes []schemas.Quote, dataType string) {
	symbol := cm.Symbol
	data := cm.Data

	if ut, ok := data[1].(string); ok {
		if ut == "hb" {
			return
		}
	}
	if upd, ok := data[1].([]interface{}); ok {
		dataType = "u"
		quotes = append(quotes, q.mapQuote(symbol, upd))
	}

	return
}

// mapQuote - mapping incoming WS message into common Quote model
func (q *QuotesGroup) mapQuote(symbol string, d []interface{}) schemas.Quote {
	smb, _, _ := parseSymbol(symbol)
	volumeBase := d[7].(float64) * d[6].(float64)

	return schemas.Quote{
		Symbol:          smb,
		Price:           strconv.FormatFloat(d[6].(float64), 'f', 8, 64),
		High:            strconv.FormatFloat(d[8].(float64), 'f', 8, 64),
		Low:             strconv.FormatFloat(d[9].(float64), 'f', 8, 64),
		DrawdownValue:   strconv.FormatFloat(d[4].(float64), 'f', 8, 64),
		DrawdownPercent: strconv.FormatFloat(d[5].(float64), 'f', 4, 64),
		VolumeBase:      strconv.FormatFloat(volumeBase, 'f', 8, 64),
		VolumeQuote:     strconv.FormatFloat(d[7].(float64), 'f', 8, 64),
	}
}
