package tidex

import (
	"fmt"
	"time"

	"github.com/syndicatedb/goex/clients"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// SymbolsProvider - order book provider
type SymbolsProvider struct {
	httpClient *clients.HTTP
}

// NewSymbolsProvider - SymbolsProvider constructor
func NewSymbolsProvider(httpProxy *proxy.Provider) *SymbolsProvider {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &SymbolsProvider{
		httpClient: clients.NewHTTP(proxyClient),
	}
}

// Get - getting all symbols from Exchange
func (sp *SymbolsProvider) Get() (symbols []schemas.Symbol, err error) {
	b, err := sp.httpClient.Get(apiSymbols, clients.Params{})
	fmt.Println("b, err: ", b, err)
	return
}

// Subscribe - getting all symbols from Exchange
func (sp *SymbolsProvider) Subscribe(d time.Duration) (ch chan schemas.Result) {
	go func() {
		for {
			b, err := sp.httpClient.Get(apiSymbols, clients.Params{})
			fmt.Println("b, err: ", b, err)
			time.Sleep(d)
		}
	}()

	return
}
