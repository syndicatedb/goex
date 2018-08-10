package binance

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/http"

	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type recentTrade struct {
	ID             int    `json:"id"`
	Price          string `json:"price"`
	Quantity       string `json:"qty"`
	Timestamp      int64  `json:"time"`
	BuyerMaker     bool   `json:"isBuyerMaker"`
	BestPriceMatch bool   `json:"isBestMatch"`
}

/*
recentTradesChannelMessage - recent trades message by ws
*/
type recentTradesChannelMessage struct {
	Type         string `json:"e"`
	Time         int64  `json:"E"` // Event time
	Symbol       string `json:"s"`
	TradeID      int    `json:"a"`
	Price        string `json:"p"`
	Quantity     string `json:"q"`
	FirstTradeID int    `json:"f"`
	LastTradeID  int    `json:"l"`
	Timestamp    int64  `json:"T"` // trade time
	IsMaker      bool   `json:"m"`
}

type recentTradesStream struct {
	Stream string                     `json:"stream"`
	Data   recentTradesChannelMessage `json:"data"`
}

// TradesGroup - trades group structure
type TradesGroup struct {
	symbols []schemas.Symbol

	wsClient   *websocket.Client
	httpClient *httpclient.Client
	httpProxy  proxy.Provider

	dataCh  chan []byte
	errorCh chan error

	resultCh chan schemas.ResultChannel
}

// NewTradesGroup - TradesGroup constructor
func NewTradesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *TradesGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &TradesGroup{
		symbols:    symbols,
		httpProxy:  httpProxy,
		httpClient: httpclient.New(proxyClient),
		dataCh:     make(chan []byte, 2*len(symbols)),
		errorCh:    make(chan error, 2*len(symbols)),
	}
}

// Start - starting updates
func (tg *TradesGroup) Start(ch chan schemas.ResultChannel) {
	log.Println("Trades starting")
	tg.resultCh = ch
	tg.listen()
	go func() {
		for _, s := range tg.symbols {
			tg.Get(s.OriginalName)
			time.Sleep(100 * time.Millisecond)
		}
	}()
	tg.connect()
}

func (tg *TradesGroup) restart() {
	tg.Start(tg.resultCh)
}

// Get - getting trades snapshot by symbol
func (tg *TradesGroup) Get(symbol string) (trades []schemas.Trade, err error) {
	var b []byte
	var resp []recentTrade

	url := apiTrades + "?" + "symbol=" + strings.ToUpper(symbol)

	if b, err = tg.httpClient.Get(url, httpclient.Params(), false); err != nil {
		log.Println("Error", err)
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}

	trades, err = tg.mapSnapshot(resp, symbol)
	tg.resultCh <- schemas.ResultChannel{
		DataType: "s",
		Data:     trades,
		Error:    err,
	}
	log.Println("Snapshot:", trades)

	return
}

func (tg *TradesGroup) mapSnapshot(data []recentTrade, symbol string) (trades []schemas.Trade, err error) {
	for _, t := range data {
		var typeStr string
		if t.BuyerMaker {
			typeStr = "buy"
		} else {
			typeStr = "sell"
		}
		price, err := strconv.ParseFloat(t.Price, 64)
		if err != nil {
			log.Println("Error mapping public trades snapshot", err)
			return nil, err
		}
		qty, err := strconv.ParseFloat(t.Quantity, 64)
		if err != nil {
			log.Println("Error mapping public trades snapshot", err)
			return nil, err
		}
		trades = append(trades, schemas.Trade{
			OrderID:   strconv.Itoa(t.ID),
			Symbol:    symbol,
			Price:     price,
			Amount:    qty,
			Timestamp: t.Timestamp,
			Type:      typeStr,
		})
	}
	return
}

// connect - creating new WS client and establishing connection
func (tg *TradesGroup) connect() {
	var smbls []string
	for _, s := range tg.symbols {
		smbls = append(smbls, strings.ToLower(s.OriginalName))
	}
	ws := websocket.NewClient(wsURL+strings.Join(smbls, "@aggTrade/")+"@aggTrade", tg.httpProxy)
	tg.wsClient = ws
	if err := tg.wsClient.Connect(); err != nil {
		log.Println("Error connecting to binance API: ", err)
		tg.restart()
	}
	tg.wsClient.Listen(tg.dataCh, tg.errorCh)
}

// listen - listening to updates from WS
func (tg *TradesGroup) listen() {
	go func() {
		for msg := range tg.dataCh {
			trades, datatype, err := tg.handleUpdates(msg)
			if len(trades) > 0 {
				tg.resultCh <- schemas.ResultChannel{
					DataType: datatype,
					Data:     trades,
					Error:    err,
				}
			}
		}
	}()
	go func() {
		for err := range tg.errorCh {
			tg.resultCh <- schemas.ResultChannel{
				Error: err,
			}
			log.Println("Error listening:", err)
			tg.restart()
		}
	}()
}

func (tg *TradesGroup) handleUpdates(data []byte) (trades []schemas.Trade, dataType string, err error) {
	var msg recentTradesStream
	err = json.Unmarshal(data, &msg)
	if err != nil {
		log.Println("Unmarshalling error:", err)
	}

	trades, err = tg.mapUpdates(msg.Data)
	if err != nil {
		log.Println("Decorating error:", err)
	}
	dataType = "u"

	return
}

func (tg *TradesGroup) mapUpdates(data recentTradesChannelMessage) (trades []schemas.Trade, err error) {
	qty, err := strconv.ParseFloat(data.Quantity, 64)
	if err != nil {
		log.Println("Error mapping trades update:", err)
		return nil, err
	}
	price, err := strconv.ParseFloat(data.Price, 64)
	if err != nil {
		log.Println("Error mapping trades update:", err)
		return nil, err
	}
	trades = append(trades, schemas.Trade{
		OrderID:   strconv.Itoa(data.TradeID),
		Symbol:    data.Symbol,
		Price:     price,
		Amount:    qty,
		Timestamp: data.Timestamp,
		Type:      data.Type,
	})

	return
}
