package schemas

import (
	"github.com/syndicatedb/goproxy/proxy"
)

// DataTypes - when sending data: update or snapshot
const (
	DataTypeSnapshot = "s"
	DataTypeUpdate   = "u"
)

// Credentials - struct to store credentials for private requests
type Credentials struct {
	APIKey    string
	APISecret string
	Sign      Signer
}

/*
Exchange - exchange struct
*/
type Exchange struct {
	symbols       []Symbol
	Credentials   Credentials
	ProxyProvider proxy.Provider

	Orders  OrdersProvider
	Symbol  SymbolProvider
	Quotes  QuotesProvider
	Trades  TradesProvider
	Candles CandlesProvider

	Trading TradingProvider
}

// OrdersProvider - getter
func (ex *Exchange) OrdersProvider() OrdersProvider {
	return ex.Orders
}

// SymbolProvider - getter
func (ex *Exchange) SymbolProvider() SymbolProvider {
	return ex.Symbol
}

// QuotesProvider - getter
func (ex *Exchange) QuotesProvider() QuotesProvider {
	return ex.Quotes
}

// TradesProvider - getter
func (ex *Exchange) TradesProvider() TradesProvider {
	return ex.Trades
}

// CandlesProvider - getter
func (ex *Exchange) CandlesProvider() CandlesProvider {
	return ex.Candles
}

// TradingProvider - getter
func (ex *Exchange) TradingProvider() TradingProvider {
	return ex.Trading
}

// Options - exchange options for init
type Options struct {
	Name          string
	API           string
	Credentials   Credentials
	ProxyProvider proxy.Provider
}
