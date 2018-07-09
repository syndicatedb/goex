package schemas

import (
	"github.com/syndicatedb/goproxy/proxy"
)

// Credentials - struct to store credentials for private requests
type Credentials struct {
	APIKey    string
	APISecret string
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
	Credentials    Credentials
	ProxyProvider  proxy.Provider
	OrdersProvider OrdersProvider
	SymbolProvider SymbolProvider
	QuotesProvider QuotesProvider
	TradesProvider TradesProvider

	UserProvider UserProvider
}

// GetOrdersProvider - getter
func (ex *Exchange) GetOrdersProvider() OrdersProvider {
	return ex.OrdersProvider
}

// GetSymbolProvider - getter
func (ex *Exchange) GetSymbolProvider() SymbolProvider {
	return ex.SymbolProvider
}

// GetQuotesProvider - getter
func (ex *Exchange) GetQuotesProvider() QuotesProvider {
	return ex.QuotesProvider
}

// GetTradesProvider - getter
func (ex *Exchange) GetTradesProvider() TradesProvider {
	return ex.TradesProvider
}

// GetUserProvider - getter
func (ex *Exchange) GetUserProvider() UserProvider {
	return ex.UserProvider
}

// Options - exchange options for init
type Options struct {
	Name          string
	Credentials   Credentials
	ProxyProvider proxy.Provider
}
