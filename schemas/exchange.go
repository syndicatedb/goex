package schemas

import (
	"github.com/syndicatedb/goproxy/proxy"
)

// Credentials - struct to store credentials for private requests
type Credentials struct {
	APIKey    string
	APISecret string
	Sign      Signer
}

// Result - sending data with channels
type Result struct {
	DataType string
	Error    error
	Data     interface{}
}

/*
Exchange - exchange struct
*/
type Exchange struct {
	symbols       []Symbol
	Credentials   Credentials
	ProxyProvider proxy.Provider

	Orders OrdersProvider
	Symbol SymbolProvider
	Quotes QuotesProvider
	Trades TradesProvider

	User    UserProvider
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

// UserProvider - getter
func (ex *Exchange) UserProvider() UserProvider {
	return ex.User
}

// TradingProvider - getter
func (ex *Exchange) TradingProvider() TradingProvider {
	return ex.Trading
}

// Options - exchange options for init
type Options struct {
	Name          string
	Credentials   Credentials
	ProxyProvider proxy.Provider
}
