package binance

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

type symbol struct {
	Symbol             string        `json:"symbol"`
	Status             string        `json:"status"`
	BaseAsset          string        `json:"baseAsset"`
	BaseAssetPrecision int           `json:"baseAssetPrecision"`
	QuoteAsset         string        `json:"quoteAsset"`
	QuotePrecision     int           `json:"quotePrecision"`
	OrderTypes         []string      `json:"orderTypes"`
	IcebergAllowed     bool          `json:"icebergAllowed"`
	Filters            []interface{} `json:"filters"`
}

type infoMessage struct {
	Timezone   string `json:"timezone"`
	ServerTime int64  `json:"serverTime"`
	RateLimits []struct {
		RateLimitType string `json:"rateLimitType"`
		Interval      string `json:"interval"`
		Limit         int    `json:"limit"`
	} `json:"rateLimits"`
	ExchangeFilters interface{} `json:"exchangeFilters"`
	Symbols         []symbol    `json:"symbols"`
}

// NewSymbolsProvider - SymbolsProvider constructor
func NewSymbolsProvider(httpProxy proxy.Provider) *SymbolsProvider {
	proxyClient := httpProxy.NewClient(exchangeName)
	return &SymbolsProvider{
		httpClient: httpclient.New(proxyClient),
	}
}

// Get - getting all symbols from Exchange
func (sp *SymbolsProvider) Get() (symbols []schemas.Symbol, err error) {
	var b []byte
	var resp infoMessage
	if b, err = sp.httpClient.Get(apiSymbols, httpclient.Params(), false); err != nil {
		return
	}
	if err = json.Unmarshal(b, &resp); err != nil {
		return
	}

	var minPrice float64
	var maxPrice float64
	var minAmount float64
	var maxAmount float64

	for _, smb := range resp.Symbols {
		name, baseCoin, quoteCoin := parseSymbol(smb.Symbol)
		for _, filter := range smb.Filters {
			if f, ok := filter.(map[string]interface{}); ok {

				if filterType, ok := f["filterType"].(string); ok && filterType == "PRICE_FILTER" {
					if min, ok := f["minPrice"].(string); ok {
						minPrice, err = strconv.ParseFloat(min, 64)
						if err != nil {
							log.Println("[BINANCE] Error parsing symbols data:", err)
						}
					}
					if max, ok := f["maxPrice"].(string); ok {
						maxPrice, err = strconv.ParseFloat(max, 64)
						if err != nil {
							log.Println("[BINANCE] Error parsing symbols data:", err)
						}
					}
				}

				if filterType, ok := f["filterType"].(string); ok && filterType == "LOT_SIZE" {
					if min, ok := f["minQty"].(string); ok {
						minAmount, err = strconv.ParseFloat(min, 64)
						if err != nil {
							log.Println("[BINANCE] Error parsing symbols data:", err)
						}
					}
					if max, ok := f["maxQty"].(string); ok {
						maxAmount, err = strconv.ParseFloat(max, 64)
						if err != nil {
							log.Println("[BINANCE] Error parsing symbols data:", err)
						}
					}
				}
			}
		}

		s := schemas.Symbol{
			Name:         name,
			OriginalName: smb.Symbol,
			Coin:         quoteCoin,
			BaseCoin:     baseCoin,
			MinPrice:     minPrice,
			MaxPrice:     maxPrice,
			MinAmount:    minAmount,
			MaxAmount:    maxAmount,
		}
		if smb.QuotePrecision > smb.BaseAssetPrecision {
			s.PricePrecision = smb.BaseAssetPrecision
		} else if smb.QuotePrecision < smb.BaseAssetPrecision {
			s.PricePrecision = smb.QuotePrecision
		} else {
			s.PricePrecision = smb.BaseAssetPrecision
		}
		s.AmountPrecision = s.PricePrecision

		symbols = append(symbols, s)
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
