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
			dch: make(chan []byte, 2*len(pairs)),
			ech: make(chan error, 2*len(pairs)),
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

// start - starting quotes updates
func (qp *QuotesProvider) start(ch chan schemas.ResultChannel) {
	qp.bus.resChannel = ch

	qp.listen()
	qp.connect()
	qp.subscribe()
}

// restart - calling start.
// Need for restarting provider on errors.
func (qp *QuotesProvider) restart() {
	qp.start(qp.bus.resChannel)
}

func (qp *QuotesProvider) connect() {
	qp.wsClient = websocket.NewClient(wsURL, qp.httpProxy)
	qp.wsClient.UsePingMessage(".")
	if err := qp.wsClient.Connect(); err != nil {
		log.Println("Error connecting to poloniex WS API: ", err)
		qp.restart()
		return
	}
	qp.wsClient.Listen(qp.bus.dch, qp.bus.ech)
}

// subscribe - subscribing to quotes updates on WS connection
func (qp *QuotesProvider) subscribe() {
	msg := tickerSubscribeMsg{
		Command: commandSubscribe,
		Channel: 1002,
	}
	if err := qp.wsClient.Write(msg); err != nil {
		log.Printf("Error subsciring to poloniex ticker")
		qp.restart()
		return
	}
}

// listen - listening to WS updates
func (qp *QuotesProvider) listen() {
	go func() {
		for msg := range qp.bus.dch {
			var data []interface{}

			// log.Printf("DATA %+v", msg)
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
			qp.restart()
			return
		}
	}()
}

// publish - publishing messages into outChannel
func (qp *QuotesProvider) publish(data interface{}, dataType string, e error) {
	qp.bus.resChannel <- schemas.ResultChannel{
		DataType: dataType,
		Data:     data,
		Error:    e,
	}
}

// mapSnapshot - mapping incoming data into common Quote model
func (qp *QuotesProvider) mapSnapshot(data map[string]quote) (quotes []schemas.Quote) {
	for symb, q := range data {
		var valueChange float64
		symbol, _, _ := parseSymbol(symb)
		lastPrice, _ := strconv.ParseFloat(q.Last, 64)
		high, _ := strconv.ParseFloat(q.HighestBid, 64)
		low, _ := strconv.ParseFloat(q.LowestAsk, 64)
		volumeBase, _ := strconv.ParseFloat(q.QuoteVolume, 64)
		volumeQuote, _ := strconv.ParseFloat(q.BaseVolume, 64)
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
			Symbol:      symbol,
			Price:       lastPrice,
			High:        high,
			Low:         low,
			ChangeRate:  percent,
			ChangeValue: valueChange,
			VolumeBase:  volumeBase,
			Volume:      volumeQuote,
		})
	}

	return
}

// mapUpdate - mapping incoming data into common Quote model
func (qp *QuotesProvider) mapUpdate(d []interface{}) schemas.Quote {
	var valueChange float64

	smb := qp.getSymbol(int(d[0].(float64)))
	if len(smb) == 0 {
		return schemas.Quote{}
	}

	symbolName, _, _ := parseSymbol(smb)
	lastPrice := parseFloat(d[1].(string))
	high := parseFloat(d[1].(string))
	low := parseFloat(d[8].(string))
	volumeBase := parseFloat(d[6].(string))
	volumeQuote := parseFloat(d[5].(string))
	percent := parseFloat(d[4].(string))
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
		Symbol:      symbolName,
		Price:       lastPrice,
		High:        high,
		Low:         low,
		ChangeValue: valueChange,
		ChangeRate:  percent,
		VolumeBase:  volumeBase,
		Volume:      volumeQuote,
	}
}

// getSymbol - loading symbol from map to match it with currencyPair ID
func (qp *QuotesProvider) getSymbol(id int) string {
	if smb, ok := qp.pairs[id]; ok {
		return smb
	}
	return ""
}

func parseFloat(s string) (d float64) {
	d, err := strconv.ParseFloat(s, 65)
	if err != nil {
		log.Println("Error parsing string to float64: ", err)
	}

	return
}
