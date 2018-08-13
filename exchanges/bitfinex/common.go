package bitfinex

import "github.com/syndicatedb/goex/schemas"

const (
	eventSubscribe  = "subscribe"
	eventSubscribed = "subscribed"
	eventInfo       = "info"

	channelOrderBook = "books"
	channelTrades    = "trades"
	channelCandles   = "candles"
	channelTicker    = "ticker"

	wsCodeStopping = 20051
)

// orderBookSubsMessage - subscription message for orderbook
type orderBookSubsMessage struct {
	Event     string `json:"event"`
	Channel   string `json:"channel"`
	Symbol    string `json:"symbol"`
	Precision string `json:"prec"`
	Frequency string `json:"freq"`
	Length    string `json:"len"`
}

// tradeSubsMessage - subscription message for trades
type tradeSubsMessage struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	Symbol  string `json:"symbol"`
}

// tickerSubsMessage - subscription message for ticker
type tickerSubsMessage struct {
	Event     string `json:"event"`
	Channel   string `json:"channel"`
	Symbol    string `json:"symbol"`
	Precision string `json:"prec"`
	Frequency string `json:"freq"`
	Length    string `json:"len"`
}

// event - Bitfinex Websocket event structure
type event struct {
	Event     string `json:"event"`
	Code      int64  `json:"code"`
	Msg       string `json:"msg"`
	Channel   string `json:"channel"`
	ChanID    int64  `json:"chanId"`
	Symbol    string `json:"symbol"`
	Precision string `json:"prec"`
	Frequency string `json:"freq"`
	Length    string `json:"len"`
	Pair      string `json:"pair"`
	Key       string `json:"key"`
}

// bus - bus of channels for groups providers.
// 'dch' is used to recieve bytes data from websocket client.
// 'ech' is used to recieve errors from websocket client.
// 'outChannel' is used to send data from providers to collector.
type bus struct {
	dch        chan []byte
	ech        chan error
	outChannel chan schemas.ResultChannel
}

func int64Value(v interface{}) int64 {
	if f, ok := v.(float64); ok {
		return int64(f)
	}
	if i, ok := v.(int64); ok {
		return i
	}
	return 0
}
