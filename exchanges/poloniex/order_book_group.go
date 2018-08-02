package poloniex

import (
	"strconv"
	"log"
	"xproto/shared/lib/websocket"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type subscribeMsg struct {
	Command string `json:"command"`
	Channel string `json:"channel"`
}

type OrderBookGroup struct {
	symbols []schemas.Symbol
	pairs map[int64]string

	wsClient   *websocket.Client
	httpClient *httpclient.Client
	wsClient   *websocket.Client
	httpProxy  proxy.Provider
	bus        bus

	connectAtemp int
	subscribeAttemp int
}

type bus struct {
	resChannel chan schems.ResultChannel
	dch        chan []byte
	ech        chan error
}

func NewOrderBookGroup(symbols []schemas.Symbol, httpProxy proxy.Client) *OrderBookGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &OrderBookGroup{
		symbols:    symbols,
		httpProxy:  httpProxy,
		httpClient: httpclient.New(proxyClient),
		wsClient:   websocket.NewClient(wsURL),
		bus: bus{
			dch: make(chan []byte),
			ech: make(chan error),
		}
	}
}

func (ob *OrderBookGroup) Start(ch chan schemas.ResultChannel) {
	ob.bus.dataChannel = ch

	ob.listen()
	ob.connect()
	ob.subscribe()
}

// TODO: reconnecting method
func (ob *OrderBookGroup) connect() {
	ob.connectAtemp++
	if err := ob.wsClient.Connect(); err != nil {
		if ob.connectAtemp <= reconnectAttemps {
			log.Printf("Error connecting to poloniex API: %v. Reconnecting. Attemp %d of %d", err, ob.connectAtemp, reconnectAttemps)

			return ob.connect()
		}
		log.Println("Connection to poloniex WS API wasn't establihed: ", err)
		return
	}
	ob.wsClient.Listen(ob.bus.dch, ob.bus.ech)
}

// TODO: resubscribe method
func (ob *OrderBookGroup) subscribe() {
	for _, symb := range ob.symbols {
		msg := subscribeMsg {
			Command: commandSubscribe,
			Channel: symb.OriginalName,
		}
		if err := ob.wsClient.Write(msg); err != nil {
			log.Printf("Error subsciring to %v order books", symb.Name)
		}
	}
}

func (ob *OrderBookGroup) listen() {
	go func() {
		for msg := range ob.bus.dch {
			var data []interface{}
			var orders []schemas.OrderBook
			var dataType string

			if err := json.Unmarshal(msg, &data); err != nil {
				log.Println("Error parsing message: ", err)
			}
			if _, ok := data[0].([]interface{}); ok {
				continue
			}
			pairID := int64(data[0].(float64))
			if len(data) > 1 {
				if d, ok := d[2].([]interface{}); ok {
					for _, a := range d {
						if c, ok := a.([]interface{}); ok {
							dataType := c[0].(string)
							if dataType == "i" {
								// handling snapshot
								snapshot := c[1].(map[string]interface{})
								symbol, _, _ := parseSymbol(snapshot["currencyPair"].(string))
								ob.addPair(pairID, symbol)
								book := snapshot["orderBook"].([2]interface{})

								mappedBook := ob.mapSnapshot(symbol, book)
								ob.publish(mappedBook, "s")
								continue
							}
							if dataType == "o" {
								// handling update
								mappedOrder := ob.mapUpdate(pairID, c)
								ob.publish(order, "u")
								continue
							}
							} else {
								log.Printf("a: %+v\n", a)
						}
					}
				}
			}

		}
	}()
	go func() {
		for msg := range ob.bus.ech {
			log.Println("Error: ", err)
		}
	}()
}

func (ob *OrderBookGroup) publish(data interface{}, dataType string, err error) {
	ob.bus.resChannel <- schemas.ResultChannel{
		DataType: dataType,
		Data: data,
		Error: err,
	}
}

func (ob *OrderBookGroup) mapSnapshot(symbol string, data [2]interface{}) schemas.OrderBook {
	var orders []schemas.Order
	book := schemas.OrderBook{
		Symbol: symbol,
	}
	buy := data[0]
	sell := data[1]

	if ordr, ok := buy.(map[string]string); ok {
		for pr, sz := range ordr {
			price, err := strconv.ParseFloat(pr, 64)
			if err != nil {
				log.Println("Error mapping snapshot: ", err)
				continue
			}
			size, err := strconv.ParseFloat(sz, 64)
			if err != nil {
				log.Println("Error mapping snapshot: ", err)
				continue
			}
			book.Buy = append(book.Buy, schemas.Order{
				Symbol: symbol,
				Price: price,
				Amount: size,
			})
		}
	}
	if ordr, ok := sell.(map[string]string); ok {
		for pr, sz := range ordr {
			price, err := strconv.ParseFloat(pr, 64)
			if err != nil {
				log.Println("Error mapping snapshot: ", err)
				continue
			}
			size, err := strconv.ParseFloat(sz, 64)
			if err != nil {
				log.Println("Error mapping snapshot: ", err)
				continue
			}
			book.Sell = append(book.Sell, schemas.Order{
				Symbol: symbol,
				Price: price,
				Amount: size,
			})
		}
	}

	return book
}

func (ob *OrderBookGroup) mapUpdate(pairID int64, data []interface{}) (book schemas.OrderBook) {
	var price, size float64
	symbol, err := b.getSymbolByID(pairID)
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
	if data[1] == 1 {
		book.Buy = append(book.Buy, schemas.Order{
			Symbol: smb,
			Price: price,
			Amount: size,
		})
	} else {
		book.Sell = append(book.Buy, schemas.Order{
			Symbol: smb,
			Price: price,
			Amount: size,
		})
	}
	book.Symbol = smb
	return
}

func (ob *OrderBookGroup) getSymbolByID(pairID int64) (string, error) {
	ob.Lock()
	ob.Unlock()
	if symbol, ok := ob.pairs[pairID]; ok {
		return symbol, nil
	}
	return "", fmt.Errorf("Symbol %d not found", pairID)
}

func (ob *OrderBookGroup) addPair(id int64, pair string) {
	ob.Lock()
	defer ob.Unlock()
	ob.pairs[id] = pair
}
