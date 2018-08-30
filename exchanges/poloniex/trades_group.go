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

type trade struct {
	ID     int64  `json:"tradeID"`
	Date   string `json:"date"`
	Type   string `json:"type"`
	Rate   string `json:"rate"`
	Amount string `json:"amount"`
	Total  string `json:"total"`
}

// TradesGroup - trade group structure
type TradesGroup struct {
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

// NewTradesGroup - TradesGroup constructor
func NewTradesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *TradesGroup {
	proxyClient := httpProxy.NewClient(exchangeName)
	pairs := currencPairs

	return &TradesGroup{
		symbols:    symbols,
		httpProxy:  httpProxy,
		httpClient: httpclient.New(proxyClient),
		pairs:      pairs,
		dch:        make(chan []byte, 2*len(symbols)),
		ech:        make(chan error, 2*len(symbols)),
	}
}

// Get - getting trades snapshot
func (tg *TradesGroup) Get() (trades [][]schemas.Trade, err error) {
	if len(tg.symbols) == 0 {
		err = errors.New("No symbols provided")
		return
	}

	for i := range tg.symbols {
		var resp []trade
		var symbol string
		var b []byte

		symbol = tg.symbols[i].OriginalName
		url := restURL

		query := httpclient.Params()
		query.Set("command", commandTrades)
		query.Set("currencyPair", symbol)

		if b, err = tg.httpClient.Get(url, query, false); err != nil {
			return
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			return
		}
		trades = append(trades, tg.mapSnapshot(symbol, resp))
		time.Sleep(1 * time.Second)
	}

	return
}

// Start - starting updates
func (tg *TradesGroup) Start(ch chan schemas.ResultChannel) {
	tg.outChannel = ch

	tg.listen()
	tg.connect()
	tg.sendSnapshot()
	tg.subscribe()
	tg.collectSnapshots()
}

func (tg *TradesGroup) restart() {
	if err := tg.wsClient.Exit(); err != nil {
		log.Println("Error destroying connection: ", err)
	}
	tg.Start(tg.outChannel)
}

func (tg *TradesGroup) connect() {
	tg.wsClient = websocket.NewClient(wsURL, tg.httpProxy)
	tg.wsClient.UsePingMessage(".")
	if err := tg.wsClient.Connect(); err != nil {
		log.Println("Error connecting to poloniex WS API: ", err)
		tg.restart()
		return
	}
	tg.wsClient.Listen(tg.dch, tg.ech)
}

func (tg *TradesGroup) subscribe() {
	for _, symb := range tg.symbols {
		msg := ordersSubscribeMsg{
			Command: commandSubscribe,
			Channel: symb.OriginalName,
		}
		if err := tg.wsClient.Write(msg); err != nil {
			log.Printf("Error subsciring to %v order books", symb.Name)
			tg.restart()
			return
		}
	}
}

// collectSnapshots getting snapshots and publishing them to outChannel
func (tg *TradesGroup) collectSnapshots() {
	go func() {
		for {
			time.Sleep(snapshotInterval)

			data, err := tg.Get()
			if err != nil {
				log.Println("Error loading trades snapshot: ", err)
			}
			for _, tr := range data {
				if len(tr) > 0 {
					tg.publish(tr, "s", nil)
				}
			}
		}
	}()
}

// listen - listening to WS channels and handle incoming messages
func (tg *TradesGroup) listen() {
	go func() {
		for msg := range tg.dch {
			var data []interface{}

			if err := json.Unmarshal(msg, &data); err != nil {
				log.Println("Error parsing message: ", err)
				continue
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
							if dataType == "t" {
								// handling trade
								mappedTrade := tg.mapUpdate(pairID, c)
								if len(mappedTrade.Symbol) > 0 {
									go tg.publish([]schemas.Trade{mappedTrade}, "u", nil)
								}
							}
						}
					}
				}
			} else {
				continue
			}
		}
	}()
	go func() {
		for err := range tg.ech {
			log.Println("Error: ", err)
			tg.restart()
			return
		}
	}()
}

// publish - publishing data into result channel
func (tg *TradesGroup) publish(data interface{}, dataType string, err error) {
	tg.outChannel <- schemas.ResultChannel{
		DataType: dataType,
		Data:     data,
		Error:    err,
	}
}

// sendSnapshot - preparing and sending snapshot into result channel
func (tg *TradesGroup) sendSnapshot() {
	trades, err := tg.Get()
	if err != nil {
		tg.publish(nil, "s", err)
	}
	for _, tr := range trades {
		tg.publish(tr, "s", nil)
	}
}

// mapSnapshot - mapping snapshot data into common trade model
func (tg *TradesGroup) mapSnapshot(symbol string, data []trade) (trades []schemas.Trade) {
	for _, tr := range data {
		var price, size float64

		layout := "2006-01-02 15:04:05"
		tms, err := time.Parse(layout, tr.Date)
		if err != nil {
			log.Println("Error parsing time: ", err)
		}

		if price, err = strconv.ParseFloat(tr.Rate, 64); err != nil {
			return
		}
		if size, err = strconv.ParseFloat(tr.Amount, 64); err != nil {
			return
		}
		smb, _, _ := parseSymbol(symbol)

		trades = append(trades, schemas.Trade{
			ID:        strconv.FormatInt(tr.ID, 10),
			Symbol:    smb,
			Type:      tr.Type,
			Price:     price,
			Amount:    size,
			Timestamp: tms.Unix(),
		})
	}

	return
}

// mapUpdate - mapping update data into common update model
func (tg *TradesGroup) mapUpdate(pairID int64, data []interface{}) schemas.Trade {
	var price, size float64
	symbol, err := tg.getSymbolByID(int(pairID))
	if err != nil {
		log.Println("Error getting symbol: ", err)
		return schemas.Trade{}
	}

	smb, _, _ := parseSymbol(symbol)
	if price, err = strconv.ParseFloat(data[4].(string), 64); err != nil {
		return schemas.Trade{}
	}
	if size, err = strconv.ParseFloat(data[3].(string), 64); err != nil {
		return schemas.Trade{}
	}

	trade := schemas.Trade{
		Symbol:    smb,
		ID:        data[1].(string),
		Price:     price,
		Amount:    size,
		Timestamp: int64(data[5].(float64)),
	}

	if int(data[2].(float64)) == 1 {
		trade.Type = schemas.Buy
	}
	if int(data[2].(float64)) == 0 {
		trade.Type = schemas.Sell
	}

	return trade
}

// getSymbolByID - getting symbol name by it's id
func (tg *TradesGroup) getSymbolByID(pairID int) (string, error) {
	if symbol, ok := tg.pairs[pairID]; ok {
		return symbol, nil
	}
	return "", fmt.Errorf("Symbol %d not found", pairID)
}
