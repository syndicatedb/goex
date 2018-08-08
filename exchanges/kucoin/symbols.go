package kucoin

import (
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type SymbolsProvider struct {
	httpClient *httpclient.Client
}

func NewSymbolsProvider(httpProxy proxy.Provider) *SymbolsProvider {
	proxyClient := httpProxy.NewClient(exchangeName)
	return &SymbolsProvider{
		httpClient: httpclient.New(proxyClient),
	}
}

func (sp *SymbolsProvider) Get() (symbols []schemas.Symbol, err error) {
	return
}

func (sp *SymbolsProvider) Subscribe(d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)
	return ch
}
