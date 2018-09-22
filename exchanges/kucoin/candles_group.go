package kucoin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

type klinesResponse struct {
	Success   string    `json:"s"`
	Close     []float64 `json:"c"`
	Volume    []float64 `json:"v"`
	Timestamp []int64   `json:"t"`
	High      []float64 `json:"h"`
	Low       []float64 `json:"l"`
	Open      []float64 `json:"o"`
}

// CandlesGroup - kucoin candles group structure
type CandlesGroup struct {
	symbols    []schemas.Symbol
	httpClient *httpclient.Client

	outChannel chan schemas.ResultChannel
}

// NewCandlesGroup - kucoin candles group constructor
func NewCandlesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *CandlesGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &CandlesGroup{
		symbols:    symbols,
		httpClient: httpclient.New(proxyClient),
	}
}

// Subscribe - starting updates for symbols
func (cg *CandlesGroup) Subscribe(ch chan schemas.ResultChannel, d time.Duration) {
	cg.outChannel = ch

	for {
		candles, err := cg.Get()
		if err != nil {
			go cg.publish(nil, "s", err)
			continue
		}
		for _, b := range candles {
			go cg.publish(b, "s", nil)
		}

		time.Sleep(d)
	}
}

// Get - loading candles snapshot by symbols
func (cg *CandlesGroup) Get() (candles [][]schemas.Candle, err error) {
	var b []byte
	var resp klinesResponse

	for _, symb := range cg.symbols {
		to := time.Now()
		from := to.Add(-1 * time.Hour)
		query := httpclient.Params()
		query.Set("symbol", symb.OriginalName)
		query.Set("from", strconv.FormatInt(from.Unix(), 10))
		query.Set("to", strconv.FormatInt(to.Unix(), 10))
		query.Set("resolution", "1")

		if b, err = cg.httpClient.Get(apiCandles, query, false); err != nil {
			return
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			return
		}
		if resp.Success != "ok" {
			err = fmt.Errorf("[KUCOIN] Error getting candle: %v", resp)
			return
		}

		candles = append(candles, cg.mapSnapshot(symb.Name, resp))
	}

	return
}

func (cg *CandlesGroup) publish(data interface{}, dataType string, e error) {
	cg.outChannel <- schemas.ResultChannel{
		DataType: dataType,
		Data:     data,
		Error:    e,
	}
}

func (cg *CandlesGroup) mapSnapshot(symbol string, data klinesResponse) (candles []schemas.Candle) {
	for i := range data.Close {
		candles = append(candles, schemas.Candle{
			Symbol:         symbol,
			Open:           data.Open[i],
			Close:          data.Close[i],
			High:           data.High[i],
			Low:            data.Low[i],
			Volume:         data.Volume[i],
			Timestamp:      data.Timestamp[i],
			Discretization: 60,
		})
	}

	return
}
