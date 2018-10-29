package poloniex

import (
	"time"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// CandlesProvider - stub poloniex candles provider
type CandlesProvider struct{}

// NewCandlesProvider - candles provider constructor
func NewCandlesProvider(httpProxy proxy.Provider) *CandlesProvider {
	return &CandlesProvider{}
}

// SetSymbols - stub method for poloniex candles provider
func (cp *CandlesProvider) SetSymbols(symbols []schemas.Symbol) schemas.CandlesProvider {
	return cp
}

// Get - stub method for poloniex candles provider
func (cp *CandlesProvider) Get(symbol schemas.Symbol) (candles []schemas.Candle, err error) {
	return
}

// Subscribe - stub method for poloniex candles provider
func (cp *CandlesProvider) Subscribe(symbol schemas.Symbol, d time.Duration) chan schemas.ResultChannel {
	return nil
}

// SubscribeAll - stub method for poloniex candles provider
func (cp *CandlesProvider) SubscribeAll(d time.Duration) chan schemas.ResultChannel {
	return nil
}
