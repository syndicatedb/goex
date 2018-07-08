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
	httpProxyProvider *proxy.Provider
	exchange          exchanges.Exchange
}

type state struct {
	symbols []schemas.Symbol
}

func NewWorker(exchangeName string, httpProxyProvider *proxy.Provider) Worker {
	return Worker{
		exchangeName:      exchangeName,
		httpProxyProvider: httpProxyProvider,
	}
}

func (w *Worker) Start() {
	var err error
	// Exchange init
	w.exchange = exchanges.NewPublic(w.exchangeName)
	w.exchange.SetProxyProvider(w.httpProxyProvider)
	w.exchange.InitProviders()
	w.state.symbols, err = w.exchange.GetSymbolProvider().Get()
	if err != nil {
		log.Fatalln("Symbols empty")
	}
	go w.orderBook(w.state.symbols)
	go w.symbols()
}

func (w *Worker) symbols() {
	chs := w.exchange.GetSymbolProvider().Subscribe(10 * time.Hour)
	for msg := range chs {
		fmt.Println("msg error: ", msg.Error)
	}
}

func (w *Worker) orderBook(symbols []schemas.Symbol) {
	chs := w.exchange.GetOrdersProvider().
		SetSymbols(symbols).
		SubscribeAll(1 * time.Second)
	for msg := range chs {
		fmt.Println("msg error: ", msg.Error)
	}
}

func (w *Worker) Exit() error {
	fmt.Println("Exit")
	return nil
}
