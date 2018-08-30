package binance

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

// CandlesGroup - bitfinex candles group structure
type CandlesGroup struct {
	symbols []schemas.Symbol

	wsClient   *websocket.Client
	httpClient *httpclient.Client
	httpProxy  proxy.Provider

	dataCh  chan []byte
	errorCh chan error

	resultCh chan schemas.ResultChannel
}

/*
klinesChannelMessage - klines message by ws
*/
type klinesChannelMessage struct {
	Type     string `json:"e"`
	Time     int64  `json:"E"`
	Symbol   string `json:"S"`
	OpenTime int64  `json:"t"`
	Kline    struct {
		Interval                 string `json:"i"`
		FirstTradeID             int64  `json:"f"`
		LastTradeID              int64  `json:"L"`
		Final                    bool   `json:"x"`
		OpenTime                 int64  `json:"t"`
		CloseTime                int64  `json:"T"`
		Open                     string `json:"o"`
		High                     string `json:"h"`
		Low                      string `json:"l"`
		Close                    string `json:"c"`
		Volume                   string `json:"v"`
		NumberOfTrades           int    `json:"n"`
		QuoteAssetVolume         string `json:"q"`
		TakerBuyBaseAssetVolume  string `json:"V"`
		TakerBuyQuoteAssetVolume string `json:"Q"`
	} `json:"k"`
}

type klinesStream struct {
	Stream string               `json:"stream"`
	Data   klinesChannelMessage `json:"data"`
}

// NewCandlesGroup - bitfinex candles group constructor
func NewCandlesGroup(symbols []schemas.Symbol, httpProxy proxy.Provider) *CandlesGroup {
	proxyClient := httpProxy.NewClient(exchangeName)

	return &CandlesGroup{
		symbols:    symbols,
		httpClient: httpclient.New(proxyClient),
	}
}

// Get - loading candles snapshot by symbol
func (cg *CandlesGroup) Get() (candles [][]schemas.Candle, err error) {
	var b []byte
	var resp []interface{}

	for _, symbol := range cg.symbols {
		url := apiKlines + "?" + "symbol=" + strings.ToUpper(symbol.OriginalName) + "&interval=1m&limit=400"

		if b, err = cg.httpClient.Get(url, httpclient.Params(), false); err != nil {
			log.Println("Error getting candles snapshot", symbol, err)
			time.Sleep(5 * time.Second)
			b, err = cg.httpClient.Get(url, httpclient.Params(), false)
			return
		}
		if err = json.Unmarshal(b, &resp); err != nil {
			log.Println("Error unmarshaling orderbook snapshot", err)
		}
		result, err := cg.mapSnapshot(resp, symbol.OriginalName)
		if err != nil {
			log.Println("Error mapping orderbook snapshot", err)
		}
		candles = append(candles, result)
	}
	return
}

// Start - starting updates
func (cg *CandlesGroup) Start(ch chan schemas.ResultChannel) {
	log.Println("Orderbook starting")
	cg.resultCh = ch

	go func() {
		for {
			result, err := cg.Get()
			cg.resultCh <- schemas.ResultChannel{
				DataType: "s",
				Data:     result,
				Error:    err,
			}
			time.Sleep(5 * time.Minute)
		}
	}()
	cg.listen()
	cg.connect()
}

func (cg *CandlesGroup) restart() {
	if err := cg.wsClient.Exit(); err != nil {
		log.Println("Error destroying connection: ", err)
	}
	cg.Start(cg.resultCh)
}

// connect - creating new WS client and establishing connection
func (cg *CandlesGroup) connect() {
	var smbls []string
	for _, s := range cg.symbols {
		smbls = append(smbls, s.OriginalName)
	}

	ws := websocket.NewClient(wsURL+strings.ToLower(strings.Join(smbls, "@kline_1m/")+"@kline_1m"), cg.httpProxy)
	cg.wsClient = ws
	if err := cg.wsClient.Connect(); err != nil {
		log.Println("Error connecting to binance API: ", err)
		cg.restart()
	}
	cg.wsClient.Listen(cg.dataCh, cg.errorCh)
}

// listen - listening to updates from WS
func (cg *CandlesGroup) listen() {
	go func() {
		for msg := range cg.dataCh {
			candles, datatype := cg.handleUpdates(msg)
			if len(candles) > 0 {
				cg.resultCh <- schemas.ResultChannel{
					DataType: datatype,
					Data:     candles,
				}
			}
		}
	}()
	go func() {
		for err := range cg.errorCh {
			cg.resultCh <- schemas.ResultChannel{
				Error: err,
			}
			log.Println("Error listening:", err)
			cg.restart()
		}
	}()
}

func (cg *CandlesGroup) handleUpdates(b []byte) (candles []schemas.Candle, dataType string) {
	var msg klinesStream
	err := json.Unmarshal(b, &msg)
	if err != nil {
		log.Println("Error handling updates", err)
		return
	}
	o, err := strconv.ParseFloat(msg.Data.Kline.Open, 64)
	if err != nil {
		log.Println("Parsing open error", err)
		return
	}
	h, err := strconv.ParseFloat(msg.Data.Kline.High, 64)
	if err != nil {
		log.Println("Parsing high error", err)
		return
	}
	l, err := strconv.ParseFloat(msg.Data.Kline.Low, 64)
	if err != nil {
		log.Println("Parsing low error", err)
		return
	}
	cl, err := strconv.ParseFloat(msg.Data.Kline.Close, 64)
	if err != nil {
		log.Println("Parsing close error", err)
		return
	}
	v, err := strconv.ParseFloat(msg.Data.Kline.Volume, 64)
	if err != nil {
		log.Println("Parsing volume error", err)
		return
	}

	c := schemas.Candle{
		Symbol:    msg.Data.Symbol,
		Timestamp: int64(msg.Data.Kline.OpenTime),
		Open:      o,
		High:      h,
		Low:       l,
		Close:     cl,
		Volume:    v,
	}
	candles = append(candles, c)

	return
}

func (cg *CandlesGroup) mapSnapshot(candles []interface{}, symbol string) (klines []schemas.Candle, err error) {
	candle := schemas.Candle{
		Symbol:         symbol,
		Discretization: 60,
	}
	for _, c := range candles {
		if k, ok := c.([12]interface{}); ok {
			if timestamp, ok := k[0].(int64); ok {
				candle.Timestamp = timestamp
			}
			if open, ok := k[1].(string); ok {
				o, err := strconv.ParseFloat(open, 64)
				if err != nil {
					return nil, err
				}
				candle.Open = o
			}
			if high, ok := k[2].(string); ok {
				h, err := strconv.ParseFloat(high, 64)
				if err != nil {
					return nil, err
				}
				candle.High = h
			}
			if low, ok := k[3].(string); ok {
				l, err := strconv.ParseFloat(low, 64)
				if err != nil {
					return nil, err
				}
				candle.Low = l
			}
			if close, ok := k[4].(string); ok {
				cl, err := strconv.ParseFloat(close, 64)
				if err != nil {
					return nil, err
				}
				candle.Close = cl
			}
			if volume, ok := k[5].(string); ok {
				v, err := strconv.ParseFloat(volume, 64)
				if err != nil {
					return nil, err
				}
				candle.Volume = v
			}

			klines = append(klines, candle)
		}
	}
	return
}
