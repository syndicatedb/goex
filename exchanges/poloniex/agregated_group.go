package poloniex

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type ordersSubscribeMsg struct {
	Command string `json:"command"`
	Channel string `json:"channel"`
}

type AgregatedGroup struct {
	symbols []schemas.Symbol
	pairs   map[int64]string

	wsClient   *websocket.Client
	httpClient *httpclient.Client
	httpProxy  proxy.Provider
	bus        bus

	sync.RWMutex
}

type bus struct {
	resChannel chan schemas.ResultChannel
	dch        chan []byte
	ech        chan error
}

func NewOrderBookGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *OrderBookGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &OrderBookGroup{
		symbols:    symbols,
		httpProxy:  httpProxy,
		httpClient: httpclient.New(proxyClient),
		pairs:      make(map[int64]string),
		bus: bus{
			dch: make(chan []byte),
			ech: make(chan error),
		},
	}
}

func (ag *AgregatedGroup) Start(ch chan schemas.ResultChannel) {
	ag.bus.resChannel = ch

	ag.listen()
	ag.connect()
	ag.subscribe()
}

// TODO: reconnect method!!!
func (ag *AgregatedGroup) connect() {
	ag.wsClient = websocket.NewClient(wsURL, ag.httpProxy)
	ag.wsClient.UsePingMessage(".")
	if err := ag.wsClient.Connect(); err != nil {
		log.Println("Error connecting to poloniex WS API: ", err)
	}
	ag.wsClient.Listen(ag.bus.dch, ag.bus.ech)
}

// TODO: resubscribe method
func (ag *AgregatedGroup) subscribe() {
	for _, symb := range ag.symbols {
		msg := ordersSubscribeMsg{
			Command: commandSubscribe,
			Channel: symb.OriginalName,
		}
		if err := ag.wsClient.Write(msg); err != nil {
			log.Printf("Error subsciring to %v order books", symb.Name)
		}
	}
}

func (ag *AgregatedGroup) listen() {
	go func() {
		for msg := range ag.bus.dch {
			var data []interface{}

			if err := json.Unmarshal(msg, &data); err != nil {
				log.Println("Error parsing message: ", err)
			}
			if _, ok := data[0].([]interface{}); ok {
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
								ag.addPair(pairID, symbol)
								book := snapshot["orderBook"].([]interface{})

								mappedBook := ag.mapSnapshot(symbol, book)
								if len(mappedBook.Buy) > 0 || len(mappedBook.Sell) > 0 {
									ag.publish(mappedBook, "s", nil)
								}
								continue
							}
							if dataType == "o" {
								// handling update
								mappedBook := ag.mapUpdate(pairID, c)
								if len(mappedBook.Buy) > 0 || len(mappedBook.Sell) > 0 {
									ag.publish(mappedBook, "u", nil)
								}
								continue
							}
						}
					}
				}
			}
		}
	}()
	go func() {
		for err := range ag.bus.ech {
			log.Println("Error: ", err)
		}
	}()
}

func (ag *AgregatedGroup) publish(data schemas.OrderBook, dataType string, err error) {
	ag.bus.resChannel <- schemas.ResultChannel{
		DataType: dataType,
		Data:     data,
		Error:    err,
	}
}

func (ag *AgregatedGroup) mapSnapshot(symbol string, data []interface{}) schemas.OrderBook {
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
				log.Println("Error mapping snapshot: ", err)
				continue
			}
			size, err := strconv.ParseFloat(sz.(string), 64)
			if err != nil {
				log.Println("Error mapping snapshot: ", err)
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
				log.Println("Error mapping snapshot: ", err)
				continue
			}
			size, err := strconv.ParseFloat(sz.(string), 64)
			if err != nil {
				log.Println("Error mapping snapshot: ", err)
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

func (ag *AgregatedGroup) mapUpdate(pairID int64, data []interface{}) (book schemas.OrderBook) {
	// log.Printf("RAW ORDERBOOK UPDATE %+v", data)
	var price, size float64
	symbol, err := ag.getSymbolByID(pairID)
	if err != nil {
		log.Println("Error getting symbol: ", err)
		return
	}

	smb, _, _ := parseSymbol(symbol)
	if price, err = strconv.ParseFloat(data[2].(string), 64); err != nil {
		return
	}
	if size, err = strconv.ParseFloat(data[3].(string), 64); err != nil {
		return
	}
	if int(data[1].(float64)) == 1 {
		book.Buy = append(book.Buy, schemas.Order{
			Symbol: smb,
			Price:  price,
			Amount: size,
		})
	}
	if int(data[1].(float64)) == 0 {
		book.Sell = append(book.Sell, schemas.Order{
			Symbol: smb,
			Price:  price,
			Amount: size,
		})
	}
	book.Symbol = smb
	return
}

func (ag *AgregatedGroup) getSymbolByID(pairID int64) (string, error) {
	ag.Lock()
	ag.Unlock()
	if symbol, ok := ag.pairs[pairID]; ok {
		return symbol, nil
	}
	return "", fmt.Errorf("Symbol %d not found", pairID)
}

func (ag *AgregatedGroup) addPair(id int64, pair string) {
	ag.Lock()
	defer ag.Unlock()
	ag.pairs[id] = pair
}
