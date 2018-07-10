package main

import (
	"fmt"
	"log"
	"time"

	"github.com/syndicatedb/goex/exchanges"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type Worker struct {
	exchangeName      string
	state             state
	httpProxyProvider proxy.Provider
	exchange          exchanges.API
}

type state struct {
	symbols []schemas.Symbol
}

func NewWorker(exchangeName string, httpProxyProvider proxy.Provider) Worker {
	return Worker{
		exchangeName:      exchangeName,
		httpProxyProvider: httpProxyProvider,
	}
}

func (w *Worker) Start() {
	var err error
	// Exchange init
	w.exchange = exchanges.New(schemas.Options{
		Name:          w.exchangeName,
		ProxyProvider: w.httpProxyProvider,
		Credentials: schemas.Credentials{
			APIKey:    "QR1MG8OT-I4C7D2WZ-30SLFZWI-FM9EK9ZM-82X1P5PO",
			APISecret: "f28d85ef232b1494b7a4f07edc1e792960ac0385c8b31cedd30e2958ffb3f859"},
	})
	w.state.symbols, err = w.exchange.SymbolProvider().Get()
	if err != nil {
		log.Fatalln("Symbols empty")
	}
	// go w.symbols()
	// w.subscribe()
	info, err := w.exchange.UserProvider().Info()
	fmt.Printf("User: %+v\n", info)
	fmt.Println("err: ", err)
}

func (w *Worker) subscribe() {
	go w.orderBook(w.state.symbols)
	go w.quotes(w.state.symbols)
	go w.trades(w.state.symbols)
}

func (w *Worker) symbols() {
	chs := w.exchange.SymbolProvider().Subscribe(10 * time.Hour)
	for msg := range chs {
		if msg.Error != nil {
			fmt.Println("Symbols error: ", msg.Error)
		}
	}
}

func (w *Worker) orderBook(symbols []schemas.Symbol) {
	chs := w.exchange.OrdersProvider().
		SetSymbols(symbols).
		SubscribeAll(1 * time.Second)
	for msg := range chs {
		if msg.Error != nil {
			fmt.Println("Order book error: ", msg.Error)
		}
	}
}
func (w *Worker) quotes(symbols []schemas.Symbol) {
	chs := w.exchange.QuotesProvider().
		SetSymbols(symbols).
		SubscribeAll(1 * time.Second)
	for msg := range chs {
		if msg.Error != nil {
			fmt.Println("Quotes error: ", msg.Error)
		}
	}
}
func (w *Worker) trades(symbols []schemas.Symbol) {
	chs := w.exchange.TradesProvider().
		SetSymbols(symbols).
		SubscribeAll(1 * time.Second)
	for msg := range chs {
		if msg.Error != nil {
			fmt.Println("Trades error: ", msg.Error)
		}
	}
}

func (w *Worker) Exit() error {
	fmt.Println("Exit")
	return nil
}
