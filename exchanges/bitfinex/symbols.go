package bitfinex

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// SymbolsProvider - order book provider
type SymbolsProvider struct {
	httpClient *httpclient.Client
}

// Symbol - bitfinex symbol model
type Symbol struct {
	Pair           string `json:"pair"`
	PricePrecision int64  `json:"price_precision"`
	InitialMargin  string `json:"initial_margin"`
	MinMargin      string `json:"minimum_margin"`
	MaxOrderSize   string `json:"maximum_order_size"`
	MinOrderSize   string `json:"minimum_order_size"`
	Expiration     string `json:"expiration"`
}

// NewSymbolsProvider - SymbolsProvider constructor
func NewSymbolsProvider(httpProxy proxy.Provider) *SymbolsProvider {
	log.Println("Constructing symbols provider")
	proxyClient := httpProxy.NewClient(exchangeName)
	return &SymbolsProvider{
		httpClient: httpclient.New(proxyClient),
	}
}

// Get - getting all symbols from Exchange
func (sp *SymbolsProvider) Get() (symbols []schemas.Symbol, err error) {
	var b []byte
	var resp []Symbol
	if b, err = sp.httpClient.Get(apiSymbols, httpclient.Params(), false); err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}

	for _, smb := range resp {
		name, coin, baseCoin := parseSymbol(smb.Pair)
		minPrice, _ := strconv.ParseFloat(smb.MinOrderSize, 64)
		maxPrice, _ := strconv.ParseFloat(smb.MaxOrderSize, 64)
		minAmount, _ := strconv.ParseFloat(smb.MinMargin, 64)

		symbols = append(symbols, schemas.Symbol{
			Name:         name,
			OriginalName: smb.Pair,
			Coin:         coin,
			BaseCoin:     baseCoin,
			MinPrice:     minPrice,
			MaxPrice:     maxPrice,
			MinAmount:    minAmount,
		})
	}

	return
}

// Subscribe - getting all symbols from Exchange
func (sp *SymbolsProvider) Subscribe(d time.Duration) chan schemas.ResultChannel {
	ch := make(chan schemas.ResultChannel)

	go func() {
		for {
			symbols, err := sp.Get()
			ch <- schemas.ResultChannel{
				Data:  symbols,
				Error: err,
			}
			time.Sleep(d)
		}
	}()
	return ch
}
