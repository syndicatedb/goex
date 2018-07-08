package tidex

import (
	"strings"

	"github.com/syndicatedb/goproxy/proxy"

	"github.com/syndicatedb/goex/schemas"
)

const (
	// URL - API endpoint
	apiSymbols   = "https://api.tidex.com/api/3/info"
	apiOrderBook = "https://api.tidex.com/api/3/depth/"
)

var (
	exchangeID   = 5
	exchangeName = "tidex"
)

/*
Tidex - exchange struct
*/
type Tidex struct {
	credentials    schemas.Credentials
	httpProxy      *proxy.Provider
	OrdersProvider schemas.OrdersProvider
	SymbolProvider schemas.SymbolProvider
}

// New - Tidex constructor. APIKey and APISecret is mandatory, but could be empty
func New(apiKey, apiSecret string) *Tidex {
	return &Tidex{
		credentials: schemas.Credentials{
			APIKey:    apiKey,
			APISecret: apiSecret,
		},
	}
}

// InitProviders - init Exchnage market and User providers
func (ex *Tidex) InitProviders() {
	ex.SymbolProvider = NewSymbolsProvider(ex.httpProxy)
	ex.OrdersProvider = NewOrdersProvider(ex.httpProxy)
}

// SetProxyProvider - setting proxy
func (ex *Tidex) SetProxyProvider(httpProxy *proxy.Provider) {
	ex.httpProxy = httpProxy
}

// GetOrdersProvider - getter
func (ex *Tidex) GetOrdersProvider() schemas.OrdersProvider {
	return ex.OrdersProvider
}

// GetSymbolProvider - getter
func (ex *Tidex) GetSymbolProvider() schemas.SymbolProvider {
	return ex.SymbolProvider
}

func parseSymbol(s string) (name, coin, baseCoin string) {
	sa := strings.Split(s, "_")
	coin = strings.ToUpper(sa[0])
	baseCoin = strings.ToUpper(sa[1])
	name = coin + "-" + baseCoin
	return
}
