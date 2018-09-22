package binance

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type Quote struct {
	Symbol          string `json:"symbol"`
	DrawdownValue   string `json:"priceChange"`
	DrawdownPercent string `json:"priceChangePercent"`
	Current         string `json:"lastPrice"`
	Open            string `json:"openPrice"`
	High            string `json:"highPrice"`
	Low             string `json:"lowPrice"`
	VolumeBase      string `json:"volume"`
	VolumeQuote     string `json:"quoteVolume"`
	Time            string `json:"closeTime"`
}

type QuotesChannelMessage struct {
	Type            string `json:"e"`
	Time            int64  `json:"E"` // Event time
	Symbol          string `json:"s"`
	DrawdownPercent string `json:"P"`
	DrawdownValue   string `json:"p"`
	BidPrice        string `json:"b"`
	AskPrice        string `json:"a"`
	Close           string `json:"c"`
	CloseTime       int64  `json:"C"`
	Open            string `json:"o"`
	OpenTime        int64  `json:"O"`
	High            string `json:"h"`
	LastTradeID     int64  `json:"L"`
	Low             string `json:"l"`
	VolumeBase      string `json:"v"`
	VolumeQuote     string `json:"q"`
}

type QuotesStream struct {
	Stream string               `json:"stream"`
	Data   QuotesChannelMessage `json:"data"`
}

// QuotesGroup - quotes group strcutre
type QuotesGroup struct {
	symbols    []schemas.Symbol
	wsClient   *websocket.Client
	httpClient *httpclient.Client
	httpProxy  proxy.Provider

	dataCh  chan []byte
	errorCh chan error

	resultCh chan schemas.ResultChannel
}

// NewQuotesGroup - QuotesGroup constructor
func NewQuotesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *QuotesGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &QuotesGroup{
		symbols:    symbols,
		httpProxy:  httpProxy,
		httpClient: httpclient.New(proxyClient),
		dataCh:     make(chan []byte, 2*len(symbols)),
		errorCh:    make(chan error, 2*len(symbols)),
	}
}

// Start - starting updates
func (q *QuotesGroup) Start(ch chan schemas.ResultChannel) {
	q.resultCh = ch
	q.listen()
	q.connect()
}

func (q *QuotesGroup) restart() {
	if err := q.wsClient.Exit(); err != nil {
		log.Println("[BINANCE] Error destroying connection: ", err)
	}
	q.Start(q.resultCh)
}

// connect - creating new WS client and establishing connection
func (q *QuotesGroup) connect() {
	var smbls []string
	for _, s := range q.symbols {
		smbls = append(smbls, strings.ToLower(s.OriginalName))
	}

	q.wsClient = websocket.NewClient(wsURL+strings.Join(smbls, "@ticker/")+"@ticker", q.httpProxy)
	if err := q.wsClient.Connect(); err != nil {
		log.Println("[BINANCE] Error connecting to binance API: ", err)
		q.restart()
	}
	q.wsClient.Listen(q.dataCh, q.errorCh)
}

// listen - listening to updates from WS
func (q *QuotesGroup) listen() {
	go func() {
		for msg := range q.dataCh {
			quotes, datatype := q.handleUpdates(msg)
			q.resultCh <- schemas.ResultChannel{
				DataType: datatype,
				Data:     quotes,
			}
		}
	}()
	go func() {
		for err := range q.errorCh {
			q.resultCh <- schemas.ResultChannel{
				Error: err,
			}
			log.Println("[BINANCE] Error listening:", err)
			q.restart()
		}
	}()
}

// Get - getting quote by one symbol
func (q *QuotesGroup) Get(symbol string) (quote schemas.Quote, err error) {
	var b []byte
	var resp Quote

	url := apiQuotes + "?" + "symbol=" + strings.ToUpper(symbol)

	if b, err = q.httpClient.Get(url, httpclient.Params(), false); err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	return q.mapQuote(resp), nil
}

func (q *QuotesGroup) handleUpdates(data []byte) (quotes schemas.Quote, dataType string) {
	var msg QuotesStream
	err := json.Unmarshal(data, &msg)
	if err != nil {
		log.Println("[BINANCE] Unmarshalling error:", err)
	}

	quotes = q.mapUpdates(msg.Data)
	if err != nil {
		log.Println("[BINANCE] Decorating error:", err)
	}
	dataType = "u"

	return
}

func (q *QuotesGroup) mapQuote(data Quote) schemas.Quote {
	price := parseFloat(data.Current)
	high := parseFloat(data.High)
	low := parseFloat(data.Low)
	ddValue := parseFloat(data.DrawdownValue)
	ddPercent := parseFloat(data.DrawdownPercent)
	volumeBase := parseFloat(data.VolumeBase)
	volumeQuote := parseFloat(data.VolumeQuote)

	return schemas.Quote{
		Symbol:      data.Symbol,
		Price:       price,
		High:        high,
		Low:         low,
		ChangeValue: ddValue,
		ChangeRate:  ddPercent,
		VolumeBase:  volumeBase,
		Volume:      volumeQuote,
	}
}

// mapQuote - mapping incoming WS message into common Quote model
func (q *QuotesGroup) mapUpdates(data QuotesChannelMessage) schemas.Quote {
	smb, _, _ := parseSymbol(data.Symbol)

	price := parseFloat(data.Close)
	high := parseFloat(data.High)
	low := parseFloat(data.Low)
	ddValue := parseFloat(data.DrawdownValue)
	ddPercent := parseFloat(data.DrawdownPercent)
	volumeBase := parseFloat(data.VolumeBase)
	volumeQuote := parseFloat(data.VolumeQuote)

	return schemas.Quote{
		Symbol:      smb,
		Price:       price,
		High:        high,
		Low:         low,
		ChangeValue: ddValue,
		ChangeRate:  ddPercent,
		VolumeBase:  volumeBase,
		Volume:      volumeQuote,
	}
}

func parseFloat(s string) (d float64) {
	d, err := strconv.ParseFloat(s, 65)
	if err != nil {
		log.Println("[BINANCE] Error parsing string to float64: ", err)
	}

	return
}
