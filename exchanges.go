package goex

import (
	"github.com/syndicatedb/goex/exchanges/binance"
	"github.com/syndicatedb/goex/exchanges/tidex"
	"github.com/syndicatedb/goex/schemas"
)

// Exchange names
const (
	Tidex    = "tidex"
	Kucoin   = "kucoin"
	Bitfinex = "bitfinex"
	Binance  = "binance"
)

// API - exchange API methods
type API interface {
	SymbolProvider() schemas.SymbolProvider
	OrdersProvider() schemas.OrdersProvider
	QuotesProvider() schemas.QuotesProvider
	TradesProvider() schemas.TradesProvider
	TradingProvider() schemas.TradingProvider
}

// New - exchange constructor
func New(opts schemas.Options) API {
	if opts.Name == Tidex {
		return tidex.New(opts)
	}
	// if opts.Name == Bitfinex {
	// 	return bitfinex.New(opts)
	// }
	if opts.Name == Binance {
		return binance.New(opts)
	}
	return nil
}
