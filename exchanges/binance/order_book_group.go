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

type Message struct {
	Stream string                  `json:"stream"`
	Data   orderbookChannelMessage `json:"data"`
}

type orderbookChannelMessage struct {
	Type          string           `json:"e"`
	Time          time.Duration    `json:"E"`
	Symbol        string           `json:"s"`
	FirstUpdateID int64            `json:"U"`
	FinalUpdateID int64            `json:"u"`
	Bids          [][3]interface{} `json:"b"`
	Asks          [][3]interface{} `json:"a"`
}

type orderBookSnapshot struct {
	LastUpdateID int64            `json:"lastUpdateId"`
	Bids         [][3]interface{} `json:"bids"`
	Asks         [][3]interface{} `json:"asks"`
}

// OrderBookGroup - order book group structure
type OrderBookGroup struct {
	symbols []schemas.Symbol

	wsClient     *websocket.Client
	httpClient   *httpclient.Client
	httpProxy    proxy.Provider
	lastUpdateID map[string]int64

	dataCh  chan []byte
	errorCh chan error

	resultCh chan schemas.ResultChannel
}

// NewOrderBookGroup - OrderBookGroup constructor
func NewOrderBookGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *OrderBookGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &OrderBookGroup{
		symbols:      symbols,
		httpProxy:    httpProxy,
		httpClient:   httpclient.New(proxyClient),
		lastUpdateID: make(map[string]int64),
		dataCh:       make(chan []byte, 2*len(symbols)),
		errorCh:      make(chan error, 2*len(symbols)),
	}
}

// Get - loading order books snapshot by one symbol
func (ob *OrderBookGroup) Get() (book []schemas.OrderBook, err error) {
	var b []byte
	var resp orderBookSnapshot
	for _, symbol := range ob.symbols {
		query := httpclient.Params()
		query.Set("symbol", strings.ToUpper(symbol.OriginalName))
		query.Set("limit", "100")

		if b, err = ob.httpClient.Get(apiOrderBook, query, false); err != nil {
			log.Println("Error getting orderbook snapshot", symbol, err)
			time.Sleep(5 * time.Second)
			return
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			log.Println("Error unmarshaling orderbook snapshot", err)
		}

		result := ob.mapSnapshot(resp, symbol.OriginalName)
		if err != nil {
			log.Println("Error mapping orderbook snapshot", err)
		}

		book = append(book, result)
	}

	return
}

// Start - starting updates
func (ob *OrderBookGroup) Start(ch chan schemas.ResultChannel) {
	log.Println("Orderbook starting")
	ob.resultCh = ch

	go func() {
		for {
			result, err := ob.Get()
			for _, book := range result {
				ob.resultCh <- schemas.ResultChannel{
					DataType: "s",
					Data:     book,
					Error:    err,
				}
			}
			time.Sleep(5 * time.Minute)
		}
	}()
	ob.listen()
	ob.connect()
}

func (ob *OrderBookGroup) restart() {
	if err := ob.wsClient.Exit(); err != nil {
		log.Println("Error destroying connection: ", err)
	}
	ob.Start(ob.resultCh)
}

// connect - creating new WS client and establishing connection
func (ob *OrderBookGroup) connect() {
	var smbls []string
	for _, s := range ob.symbols {
		smbls = append(smbls, s.OriginalName)
	}

	ws := websocket.NewClient(wsURL+strings.ToLower(strings.Join(smbls, "@depth/")+"@depth"), ob.httpProxy)
	ob.wsClient = ws
	if err := ob.wsClient.Connect(); err != nil {
		log.Println("Error connecting to binance API: ", err)
		ob.restart()
	}
	ob.wsClient.Listen(ob.dataCh, ob.errorCh)
}

// listen - listening to updates from WS
func (ob *OrderBookGroup) listen() {
	go func() {
		for msg := range ob.dataCh {
			orders, datatype := ob.handleUpdates(msg)
			if len(orders.Buy) > 0 || len(orders.Sell) > 0 {
				ob.resultCh <- schemas.ResultChannel{
					DataType: datatype,
					Data:     orders,
				}
			}
		}
	}()
	go func() {
		for err := range ob.errorCh {
			ob.resultCh <- schemas.ResultChannel{
				Error: err,
			}
			log.Println("Error listening:", err)
			ob.restart()
		}
	}()
}

// handleMessage - handling message from WS
func (ob *OrderBookGroup) handleUpdates(data []byte) (orders schemas.OrderBook, dataType string) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		log.Println("Error handling updates", err)
	}

	if msg.Data.FinalUpdateID <= ob.lastUpdateID[msg.Data.Symbol] {
		return
	}

	result := ob.mapUpdates(msg.Data)
	dataType = "u"

	return result, dataType
}

// TODO: optimize this code
func (ob *OrderBookGroup) mapSnapshot(data orderBookSnapshot, symbol string) schemas.OrderBook {
	smb, _, _ := parseSymbol(symbol)
	orderBook := schemas.OrderBook{
		Symbol: smb,
	}

	for _, bid := range data.Bids {
		price, err := strconv.ParseFloat(bid[0].(string), 64)
		if err != nil {
			log.Println("Error mapping public orderbook snapshot", err)

		}
		amount, err := strconv.ParseFloat(bid[1].(string), 64)
		if err != nil {
			log.Println("Error mapping public orderbook snapshot", err)
		}
		buy := schemas.Order{
			Symbol: smb,
			Type:   schemas.Buy,
			Price:  price,
			Amount: amount,
		}
		if amount == 0 {
			buy.Remove = 1
		}
		orderBook.Buy = append(orderBook.Buy, buy)
	}

	for _, ask := range data.Asks {
		price, err := strconv.ParseFloat(ask[0].(string), 64)
		if err != nil {
			log.Println("Error mapping public orderbook snapshot", err)
		}
		amount, err := strconv.ParseFloat(ask[1].(string), 64)
		if err != nil {
			log.Println("Error mapping public orderbook snapshot", err)
		}
		sell := schemas.Order{
			Symbol: smb,
			Type:   schemas.Sell,
			Price:  price,
			Amount: amount,
		}
		if amount == 0 {
			sell.Remove = 1
		}
		orderBook.Sell = append(orderBook.Sell, sell)
	}

	return orderBook
}

// mapOrderBook - mapping incoming books message into commot OrderBook model
func (ob *OrderBookGroup) mapUpdates(data orderbookChannelMessage) schemas.OrderBook {
	smb, _, _ := parseSymbol(data.Symbol)
	orderBook := schemas.OrderBook{
		Symbol: smb,
	}

	for _, bid := range data.Bids {
		price, err := strconv.ParseFloat(bid[0].(string), 64)
		if err != nil {
			log.Println("Error mapping public orderbook snapshot", err)
		}
		amount, err := strconv.ParseFloat(bid[1].(string), 64)
		if err != nil {
			log.Println("Error mapping public orderbook snapshot", err)
		}
		buy := schemas.Order{
			Symbol: smb,
			Type:   schemas.Buy,
			Price:  price,
			Amount: amount,
		}
		if amount == 0 {
			buy.Remove = 1
		}
		orderBook.Buy = append(orderBook.Buy, buy)
	}

	for _, ask := range data.Asks {
		price, err := strconv.ParseFloat(ask[0].(string), 64)
		if err != nil {
			log.Println("Error mapping public orderbook snapshot", err)
		}
		amount, err := strconv.ParseFloat(ask[1].(string), 64)
		if err != nil {
			log.Println("Error mapping public orderbook snapshot", err)
		}
		sell := schemas.Order{
			Symbol: smb,
			Type:   schemas.Sell,
			Price:  price,
			Amount: amount,
		}
		if amount == 0 {
			sell.Remove = 1
		}
		orderBook.Sell = append(orderBook.Sell, sell)
	}

	return orderBook
}
