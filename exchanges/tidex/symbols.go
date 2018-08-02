package tidex

import (
	"encoding/json"
	"log"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// SymbolsProvider - order book provider
type SymbolsProvider struct {
	httpClient *httpclient.Client
}

// NewSymbolsProvider - SymbolsProvider constructor
func NewSymbolsProvider(httpProxy proxy.Provider) *SymbolsProvider {
	proxyClient := httpProxy.NewClient(exchangeName)
	return &SymbolsProvider{
		httpClient: httpclient.New(proxyClient),
	}
}

// Get - getting all symbols from Exchange
func (sp *SymbolsProvider) Get() (symbols []schemas.Symbol, err error) {
	var b []byte
	if b, err = sp.httpClient.Get(apiSymbols, httpclient.Params(), false); err != nil {
		return
	}
	var resp SymbolResponse
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	for sname, d := range resp.Pairs {
		if d.Hidden == 0 {
			name, coin, baseCoin := parseSymbol(sname)
			symbols = append(symbols, schemas.Symbol{
				Name:         name,
				OriginalName: sname,
				Coin:         coin,
				BaseCoin:     baseCoin,
				Fee:          d.Fee,
				MinPrice:     d.MinPrice,
				MaxPrice:     d.MaxPrice,
				MinAmount:    d.MinAmount,
				MaxAmount:    d.MaxAmount,
			})
		}
	}
	return
}

// Subscribe - getting all symbols from Exchange
func (sp *SymbolsProvider) Subscribe(d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)

	go func() {
		for {
			symbols, err := sp.Get()
			ch <- schemas.ResultChannel{
				Data:  symbols,
				Error: err,
			}
			time.Sleep(d)
		}
	}()
	return ch
}

func (b *Books) listen() {
	go func() {
		for msg := range b.dch {
			var data []interface{}
			var trades []schemas.Trade
			var orders []schemas.BookOrder
			var dtypeBook, dtypeTrade string
			if err := json.Unmarshal(msg, &data); err != nil {
				log.Println("Error parsing message: ", err)
				log.Printf("Message: %+v\n", string(msg))
				continue
			}
			if _, ok := data[0].([]interface{}); ok {
				log.Printf("data: %+v\n", data)
				continue
			}
			pairID := int64(data[0].(float64))
			if len(data) > 1 {
				d := data[2].([]interface{})
				for _, a := range d {
					if c, ok := a.([]interface{}); ok {
						dataType := c[0].(string)
						if dataType == "i" {
							i := c[1].(map[string]interface{})
							symbol := invertSymbol(i["currencyPair"].(string))
							b.addPair(pairID, symbol)
							ob := i["orderBook"].([]interface{})
							o := b.publishSnapshot(symbol, 1, ob[0].(map[string]interface{}))
							orders = append(orders, o...)
							o = b.publishSnapshot(symbol, -1, ob[1].(map[string]interface{}))
							orders = append(orders, o...)
							dtypeBook = "snapshot"
							continue
						}
						if dataType == "o" {
							order := b.publishOrder(pairID, c)
							orders = append(orders, order)
							dtypeBook = "update"
							continue
						}
						if dataType == "t" {
							// trades!
							trades = append(trades, b.parseTrade(pairID, c))
							dtypeTrade = "update"
							continue
						}
						log.Printf("Type unknown: %+v — %+v\n", pairID, c)
					} else {
						log.Printf("a: %+v\n", a)
					}
				}
				if len(orders) > 0 {
					data := [2]interface{}{orders, dtypeBook}
					b.out <- data
				}
				if len(trades) > 0 {
					data := [2]interface{}{trades, dtypeTrade}
					b.outTrades <- data
				}
			}
		}
	}()
	go func() {
		for msg := range b.ech {
			log.Println("error: ", msg)
		}
	}()
}
