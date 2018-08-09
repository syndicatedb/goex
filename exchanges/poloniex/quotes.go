package poloniex

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type tickerSubscribeMsg struct {
	Command string `json:"command"`
	Channel int    `json:"channel"`
}

type quote struct {
	Last          string `json:"last"`
	LowestAsk     string `json:"lowestAsk"`
	HighestBid    string `json:"highestBid"`
	PercentChange string `json:"percentChange"`
	BaseVolume    string `json:"baseVolume"`
	QuoteVolume   string `json:"quoteVolume"`
}

// QuotesProvider - quotes provider structure
type QuotesProvider struct {
	wsClient   *websocket.Client
	httpClient *httpclient.Client
	httpProxy  proxy.Provider
	bus        bus

	pairs map[int]string
}

// NewQuotesProvider - QuotesProvider constructor
func NewQuotesProvider(httpProxy proxy.Provider) *QuotesProvider {
	proxyClient := httpProxy.NewClient(exchangeName)
	pairs := currencPairs

	return &QuotesProvider{
		httpProxy:  httpProxy,
		httpClient: httpclient.New(proxyClient),
		pairs:      pairs,
		bus: bus{
			dch: make(chan []byte),
			ech: make(chan error),
		},
	}
}

// SetSymbols - setting symbols and creating groups by symbols chunks
func (qp *QuotesProvider) SetSymbols(symbols []schemas.Symbol) schemas.QuotesProvider {
	return qp
}

// Get - getting quotes by symbol
func (qp *QuotesProvider) Get(symbol schemas.Symbol) (q schemas.Quote, err error) {
	var b []byte
	var resp map[string]quote

	query := httpclient.Params()
	query.Set("command", commandTicker)

	if b, err = qp.httpClient.Get(restURL, query, false); err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	ticks := qp.mapSnapshot(resp)
	for _, t := range ticks {
		if t.Symbol == symbol.Name {
			return t, nil
		}
	}

	err = fmt.Errorf("No quotes found for %v", symbol.Name)
	return
}

// Subscribe - subscribing to quote by one symbol
func (qp *QuotesProvider) Subscribe(symbol schemas.Symbol, d time.Duration) chan schemas.ResultChannel {
	return qp.SubscribeAll(d)
}

// SubscribeAll - subscribing to all quotes with interval
func (qp *QuotesProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	bufLength := len(qp.pairs)
	ch := make(chan schemas.ResultChannel, 2*bufLength)
	go qp.start(ch)
	return ch
}

func (qp *QuotesProvider) start(ch chan schemas.ResultChannel) {
	qp.bus.resChannel = ch

	qp.listen()
	qp.connect()
	qp.subscribe()
}

// TODO: reconnect method
func (qp *QuotesProvider) connect() {
	qp.wsClient = websocket.NewClient(wsURL, qp.httpProxy)
	qp.wsClient.UsePingMessage(".")
	if err := qp.wsClient.Connect(); err != nil {
		log.Println("Error connecting to poloniex WS API: ", err)
	}
	qp.wsClient.Listen(qp.bus.dch, qp.bus.ech)
}

// TODO: resubscribe method
func (qp *QuotesProvider) subscribe() {
	msg := tickerSubscribeMsg{
		Command: commandSubscribe,
		Channel: 1002,
	}
	if err := qp.wsClient.Write(msg); err != nil {
		log.Printf("Error subsciring to poloniex ticker")
	}
}

func (qp *QuotesProvider) listen() {
	go func() {
		for msg := range qp.bus.dch {
			var data []interface{}

			log.Printf("DATA %+v", msg)
			if err := json.Unmarshal(msg, &data); err != nil {
				log.Println("Error parsing message:", err)
				continue
			}
			if len(data) > 2 {
				for i := 2; i < len(data); i++ {
					if t, ok := data[i].([]interface{}); ok {
						mappedQuote := qp.mapUpdate(t)
						if len(mappedQuote.Symbol) > 0 {
							qp.publish(mappedQuote, "u", nil)
						}
					}
				}
			}
		}
	}()

	go func() {
		for err := range qp.bus.ech {
			log.Println("Error: ", err)
		}
	}()
}

func (qp *QuotesProvider) publish(data interface{}, dataType string, e error) {
	qp.bus.resChannel <- schemas.ResultChannel{
		DataType: dataType,
		Data:     data,
		Error:    e,
	}
}

func (qp *QuotesProvider) mapSnapshot(data map[string]quote) (quotes []schemas.Quote) {
	for symb, q := range data {
		var valueChange float64
		symbol, _, _ := parseSymbol(symb)
		lastPrice, _ := strconv.ParseFloat(q.Last, 64)
		percent, _ := strconv.ParseFloat(q.PercentChange, 64)
		percentChange := math.Abs(percent)
		if percent > 0 {
			valueChange = lastPrice - ((lastPrice * (100 - percentChange)) / 100.00)
		}
		if percent < 0 {
			valueChange = -(((lastPrice * (100 + percentChange)) / 100.00) - lastPrice)
		}
		if percent == 0 {
			valueChange = 0
		}

		quotes = append(quotes, schemas.Quote{
			Symbol:          symbol,
			Price:           q.Last,
			High:            q.HighestBid,
			Low:             q.LowestAsk,
			DrawdownPercent: q.PercentChange,
			DrawdownValue:   strconv.FormatFloat(valueChange, 'f', 8, 64),
			VolumeBase:      q.QuoteVolume,
			VolumeQuote:     q.BaseVolume,
		})
	}

	return
}

func (qp *QuotesProvider) mapUpdate(d []interface{}) schemas.Quote {
	var valueChange float64

	smb := qp.getSymbol(int(d[0].(float64)))
	if len(smb) == 0 {
		return schemas.Quote{}
	}

	symbolName, _, _ := parseSymbol(smb)
	lastPrice, _ := strconv.ParseFloat(d[1].(string), 64)
	percent, _ := strconv.ParseFloat(d[4].(string), 64)
	percentChange := math.Abs(percent)
	if percent > 0 {
		valueChange = lastPrice - ((lastPrice * (100 - percentChange)) / 100.00)
	}
	if percent < 0 {
		valueChange = -(((lastPrice * (100 + percentChange)) / 100.00) - lastPrice)
	}
	if percent == 0 {
		valueChange = 0
	}

	return schemas.Quote{
		Symbol:          symbolName,
		Price:           d[1].(string),
		High:            d[8].(string),
		Low:             d[9].(string),
		DrawdownValue:   strconv.FormatFloat(valueChange, 'f', 8, 64),
		DrawdownPercent: d[4].(string),
		VolumeBase:      d[6].(string),
		VolumeQuote:     d[5].(string),
	}
}

func (qp *QuotesProvider) getSymbol(id int) string {
	if smb, ok := qp.pairs[id]; ok {
		return smb
	}
	return ""
}
