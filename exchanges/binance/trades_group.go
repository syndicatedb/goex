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
	tg.resultCh = ch
	tg.listen()
	go func() {
		for {
			result, err := tg.Get()
			tg.resultCh <- schemas.ResultChannel{
				DataType: "s",
				Data:     result,
				Error:    err,
			}
			time.Sleep(5 * time.Minute)
		}
	}()
	tg.connect()
}

// Stop closes WS connection
func (tg *TradesGroup) Stop() error {
	return tg.wsClient.Exit()
}

func (tg *TradesGroup) restart() {
	time.Sleep(5 * time.Second)
	if err := tg.wsClient.Exit(); err != nil {
		log.Println("[BINANCE] Error destroying connection: ", err)
	}
	tg.Start(tg.resultCh)
}

// Get - getting trades snapshot by symbol
func (tg *TradesGroup) Get() (result [][]schemas.Trade, err error) {
	var b []byte
	var trades []schemas.Trade
	for _, symbol := range tg.symbols {
		var resp []recentTrade

		url := apiTrades + "?" + "symbol=" + strings.ToUpper(symbol.OriginalName) + "&limit=200"

		if b, err = tg.httpClient.Get(url, httpclient.Params(), false); err != nil {
			log.Println("Error", err)
			return
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			return
		}

		trades, err = tg.mapSnapshot(resp, symbol.OriginalName)
		if err != nil {
			log.Println("[BINANCE] Error mapping trades snapshot", err)
		}

		result = append(result, trades)
	}

	return
}

func (tg *TradesGroup) mapSnapshot(data []recentTrade, symbol string) (trades []schemas.Trade, err error) {
	for _, t := range data {
		var typeStr string
		if t.BuyerMaker {
			typeStr = schemas.Buy
		} else {
			typeStr = schemas.Sell
		}
		price, err := strconv.ParseFloat(t.Price, 64)
		if err != nil {
			log.Println("[BINANCE] Error mapping public trades snapshot", err)
			return nil, err
		}
		qty, err := strconv.ParseFloat(t.Quantity, 64)
		if err != nil {
			log.Println("[BINANCE] Error mapping public trades snapshot", err)
			return nil, err
		}
		symb, _, _ := parseSymbol(symbol)
		trades = append(trades, schemas.Trade{
			ID:        strconv.Itoa(t.ID),
			Symbol:    symb,
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
		log.Println("[BINANCE] Error connecting to binance API: ", err)
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
			log.Println("[BINANCE] Error listening:", err)
			tg.restart()
		}
	}()
}

func (tg *TradesGroup) handleUpdates(data []byte) (trades []schemas.Trade, dataType string, err error) {
	var msg recentTradesStream
	err = json.Unmarshal(data, &msg)
	if err != nil {
		log.Println("[BINANCE] Unmarshalling error:", err)
	}

	trades, err = tg.mapUpdates(msg.Data)
	if err != nil {
		log.Println("[BINANCE] Decorating error:", err)
	}
	dataType = "u"

	return
}

func (tg *TradesGroup) mapUpdates(data recentTradesChannelMessage) (trades []schemas.Trade, err error) {
	qty, err := strconv.ParseFloat(data.Quantity, 64)
	if err != nil {
		log.Println("[BINANCE] Error mapping trades update:", err)
		return nil, err
	}
	price, err := strconv.ParseFloat(data.Price, 64)
	if err != nil {
		log.Println("[BINANCE] Error mapping trades update:", err)
		return nil, err
	}
	var typeStr string
	if data.IsMaker {
		typeStr = schemas.Buy
	} else {
		typeStr = schemas.Sell
	}
	symb, _, _ := parseSymbol(data.Symbol)
	trades = append(trades, schemas.Trade{
		ID:        strconv.Itoa(data.TradeID),
		Symbol:    symb,
		Price:     price,
		Amount:    qty,
		Timestamp: data.Timestamp,
		Type:      typeStr,
	})

	return
}
