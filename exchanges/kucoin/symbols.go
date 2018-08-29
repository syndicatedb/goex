package kucoin

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type symbolsResponse struct {
	responseHeader
	Data []symbol `json:"data"`
}

type coinsResponse struct {
	responseHeader
	Data []coin `json:"data"`
}

// SymbolsProvider structure
type SymbolsProvider struct {
	httpClient *httpclient.Client
}

// NewSymbolsProvider  - SymbolsProvider constructor
func NewSymbolsProvider(httpProxy proxy.Provider) *SymbolsProvider {
	proxyClient := httpProxy.NewClient(exchangeName)
	return &SymbolsProvider{
		httpClient: httpclient.New(proxyClient),
	}
}

// Get Loading symbols and coins and returning symbols with pricePrecision
func (sp *SymbolsProvider) Get() (symbols []schemas.Symbol, err error) {
	smbls, err := sp.getSymbols()
	if err != nil {
		return
	}
	coins, err := sp.getCoins()
	if err != nil {
		return
	}

	for _, smb := range smbls {
		var basePrec, quotePrec int

		s := smb.Map()
		if p, ok := coins[smb.CoinType]; ok {
			basePrec = int(p.TradePrecision)
		}
		if p, ok := coins[smb.CoinTypePair]; ok {
			quotePrec = int(p.TradePrecision)
		}

		if basePrec != 0 && quotePrec != 0 {
			if basePrec > quotePrec {
				s.PricePrecision = quotePrec
			} else if quotePrec > basePrec {
				s.PricePrecision = basePrec
			} else {
				s.PricePrecision = basePrec
			}
		} else {
			s.PricePrecision = defaultPrecision
		}

		symbols = append(symbols, s)
	}

	return
}

// getSymbols making http request and loading symbols data from exchange
func (sp *SymbolsProvider) getSymbols() (symbols []symbol, err error) {
	var b []byte
	var resp symbolsResponse
	if b, err = sp.httpClient.Get(apiSymbols, httpclient.Params(), false); err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if !resp.Success {
		err = fmt.Errorf("Error getting symbols: %v", resp.Message)
		return
	}
	for _, smb := range resp.Data {
		symbols = append(symbols, smb)
	}

	return
}

// getCoins making http request and loading сщшты data from exchange
func (sp *SymbolsProvider) getCoins() (coins map[string]coin, err error) {
	var b []byte
	var resp coinsResponse

	coins = make(map[string]coin)

	if b, err = sp.httpClient.Get(apiCoins, httpclient.Params(), false); err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}
	if !resp.Success {
		err = fmt.Errorf("Error getting coins: %v", resp.Message)
		return
	}
	for _, coin := range resp.Data {
		coins[coin.Coin] = coin
	}

	return
}

// Subscribe - subscribing to symbols updates with period 'd'
func (sp *SymbolsProvider) Subscribe(d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)

	go func() {
		for {
			symbols, err := sp.Get()
			ch <- schemas.ResultChannel{
				DataType: "s",
				Data:     symbols,
				Error:    err,
			}

			time.Sleep(d)
		}
	}()

	return ch
}
