package poloniex

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type ordersSubscribeMsg struct {
	Command string `json:"command"`
	Channel string `json:"channel"`
}

type orderbook struct {
	Asks     [][]interface{} `json:"asks"`
	Bids     [][]interface{} `json:"bids"`
	IsFrozen string          `json:"isFrozen"`
	Seq      int64           `json:"seq"`
}

// OrderBookGroup - order book group structure
type OrderBookGroup struct {
	symbols []schemas.Symbol
	pairs   map[int]string

	wsClient   *websocket.Client
	httpClient *httpclient.Client
	httpProxy  proxy.Provider

	outChannel chan schemas.ResultChannel
	dch        chan []byte
	ech        chan error
	// bus        bus
}

type bus struct {
	resChannel chan schemas.ResultChannel
	dch        chan []byte
	ech        chan error
}

// NewOrderBookGroup - OrderBookGroup constructor
func NewOrderBookGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *OrderBookGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &OrderBookGroup{
		symbols:    symbols,
		httpProxy:  httpProxy,
		httpClient: httpclient.New(proxyClient),
		pairs:      currencPairs,
		dch:        make(chan []byte, 2*len(symbols)),
		ech:        make(chan error, 2*len(symbols)),
	}
}

// Get - getting orderbook snapshot
func (ob *OrderBookGroup) Get() (books []schemas.OrderBook, err error) {
	var b []byte
	var resp orderbook
	if len(ob.symbols) == 0 {
		err = errors.New("[POLONIEX] Symbol is empty")
		return
	}

	for _, symb := range ob.symbols {
		symbol := symb.OriginalName
		query := httpclient.Params()
		query.Set("command", commandOrderBook)
		query.Set("currencyPair", symbol)
		query.Set("depth", "200")

		if b, err = ob.httpClient.Get(restURL, query, false); err != nil {
			return
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			return
		}
		books = append(books, ob.mapHTTPSnapshot(symb.Name, resp))
		time.Sleep(1 * time.Second)
	}

	return
}

// Start - starting updates
func (ob *OrderBookGroup) Start(ch chan schemas.ResultChannel) {
	ob.outChannel = ch

	ob.listen()
	ob.connect()
	ob.subscribe()
	ob.collectSnapshots()
}

func (ob *OrderBookGroup) restart() {
	time.Sleep(5 * time.Second)
	if err := ob.wsClient.Exit(); err != nil {
		log.Println("[POLONIEX] Error destroying connection: ", err)
	}
	ob.Start(ob.outChannel)
}

func (ob *OrderBookGroup) connect() {
	ob.wsClient = websocket.NewClient(wsURL, ob.httpProxy)
	ob.wsClient.UsePingMessage(".")
	if err := ob.wsClient.Connect(); err != nil {
		log.Println("[POLONIEX] Error connecting to poloniex WS API: ", err)
		ob.restart()
		return
	}
	ob.wsClient.Listen(ob.dch, ob.ech)
}

func (ob *OrderBookGroup) subscribe() {
	for _, symb := range ob.symbols {
		msg := ordersSubscribeMsg{
			Command: commandSubscribe,
			Channel: symb.OriginalName,
		}
		if err := ob.wsClient.Write(msg); err != nil {
			log.Printf("[POLONIEX] Error subsciring to %v order books", symb.Name)
			ob.restart()
			return
		}
	}
	log.Println("[POLONIEX] Subscription ok")
}

// collectSnapshots getting snapshots and publishing them to outChannel
func (ob *OrderBookGroup) collectSnapshots() {
	go func() {
		for {
			time.Sleep(snapshotInterval)

			data, err := ob.Get()
			if err != nil {
				log.Println("[POLONIEX] Error loading orderbook snapshot: ", err)
			}
			for _, book := range data {
				if len(book.Buy) > 0 || len(book.Sell) > 0 {
					ob.publish(book, "s", nil)
				}
			}
		}
	}()
}

func (ob *OrderBookGroup) listen() {
	log.Println("[POLONIEX] Start listening")
	go func() {
		for msg := range ob.dch {
			var data []interface{}

			if err := json.Unmarshal(msg, &data); err != nil {
				log.Println("[POLONIEX] Error parsing message: ", err)
				continue
			}
			if _, ok := data[0].([]interface{}); ok {
				// log.Printf("data: %+v\n", data)
				continue
			}
			pairID := int64(data[0].(float64))
			if len(data) > 1 {
				if d, ok := data[2].([]interface{}); ok {
					for _, a := range d {
						if c, ok := a.([]interface{}); ok {
							dataType := c[0].(string)
							if dataType == "i" {
								// handling snapshot
								snapshot := c[1].(map[string]interface{})
								symbol, _, _ := parseSymbol(snapshot["currencyPair"].(string))
								book := snapshot["orderBook"].([]interface{})

								mappedBook := ob.mapSnapshot(symbol, book)
								if len(mappedBook.Buy) > 0 || len(mappedBook.Sell) > 0 {
									go ob.publish(mappedBook, "s", nil)
								}
							}
							if dataType == "o" {
								// handling update
								mappedBook := ob.mapUpdate(pairID, c)
								if len(mappedBook.Buy) > 0 || len(mappedBook.Sell) > 0 {
									go ob.publish(mappedBook, "u", nil)
								}
								continue
							}
						}
					}
				} else {
					continue
				}
			} else {
				continue
			}
		}
	}()
	go func() {
		for msg := range ob.ech {
			log.Println("[POLONIEX] Error: ", msg)
			ob.restart()
			return
		}
	}()
}

func (ob *OrderBookGroup) publish(data schemas.OrderBook, dataType string, err error) {
	ob.outChannel <- schemas.ResultChannel{
		DataType: dataType,
		Data:     data,
		Error:    err,
	}
}

func (ob *OrderBookGroup) mapSnapshot(symbol string, data []interface{}) schemas.OrderBook {
	var buy, sell interface{}
	book := schemas.OrderBook{
		Symbol: symbol,
	}

	if len(data) == 2 {
		if data[0] != nil {
			buy = data[0]
		}
		if data[1] != nil {
			sell = data[1]
		}
	} else {
		return schemas.OrderBook{}
	}

	if ordr, ok := buy.(map[string]interface{}); ok {
		for pr, sz := range ordr {
			price, err := strconv.ParseFloat(pr, 64)
			if err != nil {
				log.Println("[POLONIEX] Error mapping snapshot: ", err)
				continue
			}
			size, err := strconv.ParseFloat(sz.(string), 64)
			if err != nil {
				log.Println("[POLONIEX] Error mapping snapshot: ", err)
				continue
			}
			book.Buy = append(book.Buy, schemas.Order{
				Symbol: symbol,
				Price:  price,
				Amount: size,
			})
		}
	}
	if ordr, ok := sell.(map[string]interface{}); ok {
		for pr, sz := range ordr {
			price, err := strconv.ParseFloat(pr, 64)
			if err != nil {
				log.Println("[POLONIEX] Error mapping snapshot: ", err)
				continue
			}
			size, err := strconv.ParseFloat(sz.(string), 64)
			if err != nil {
				log.Println("[POLONIEX] Error mapping snapshot: ", err)
				continue
			}
			book.Sell = append(book.Sell, schemas.Order{
				Symbol: symbol,
				Price:  price,
				Amount: size,
			})
		}
	}

	return book
}

func (ob *OrderBookGroup) mapUpdate(pairID int64, data []interface{}) (book schemas.OrderBook) {
	var price, size float64

	remove := 0
	symbol, err := ob.getSymbolByID(pairID)
	if err != nil {
		log.Println("[POLONIEX] Error getting symbol: ", err)
		return
	}

	smb, _, _ := parseSymbol(symbol)
	if price, err = strconv.ParseFloat(data[2].(string), 64); err != nil {
		return
	}
	if size, err = strconv.ParseFloat(data[3].(string), 64); err != nil {
		return
	}
	if size == 0 {
		remove = 1
	}

	if int(data[1].(float64)) == 1 {
		book.Buy = append(book.Buy, schemas.Order{
			Symbol: smb,
			Price:  price,
			Amount: size,
			Remove: remove,
		})
	}
	if int(data[1].(float64)) == 0 {
		book.Sell = append(book.Sell, schemas.Order{
			Symbol: smb,
			Price:  price,
			Amount: size,
			Remove: remove,
		})
	}
	book.Symbol = smb
	return
}

func (ob *OrderBookGroup) mapHTTPSnapshot(symbol string, data orderbook) schemas.OrderBook {
	book := schemas.OrderBook{
		Symbol: symbol,
	}

	for _, asks := range data.Asks {
		price, err := strconv.ParseFloat(asks[0].(string), 10)
		if err != nil {
			log.Println("[POLONIEX] Error mapping orderbook snapshot: ", err)
		}
		book.Sell = append(book.Sell, schemas.Order{
			Symbol: symbol,
			Price:  price,
			Amount: asks[1].(float64),
		})
	}
	for _, bids := range data.Bids {
		price, err := strconv.ParseFloat(bids[0].(string), 10)
		if err != nil {
			log.Println("[POLONIEX] Error mapping orderbook snapshot: ", err)
		}
		book.Buy = append(book.Buy, schemas.Order{
			Symbol: symbol,
			Price:  price,
			Amount: bids[1].(float64),
		})
	}

	return book
}

func (ob *OrderBookGroup) getSymbolByID(pairID int64) (string, error) {
	if symbol, ok := ob.pairs[int(pairID)]; ok {
		return symbol, nil
	}
	return "", fmt.Errorf("[POLONIEX] Symbol %d not found", pairID)
}
