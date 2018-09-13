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

// candlesSubsMessage - subscribing message for candles
type candlesSubsMessage struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	Key     string `json:"key"`
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

// authMsg represents trading data auth message model
type authMsg struct {
	Event       string `json:"event"`
	APIKey      string `json:"apiKey"`
	AuthSig     string `json:"authSig"`
	AuthPayload string `json:"authPayload"`
	AuthNonce   string `json:"authNonce"`
}

// accessResponse represents user account access info response
type accessResponse struct {
	Account   accessFlags `json:"account"`
	History   accessFlags `json:"history"`
	Orders    accessFlags `json:"orders"`
	Positions accessFlags `json:"positions"`
	Funding   accessFlags `json:"funding"`
	Wallets   accessFlags `json:"wallets"`
	Withdraw  accessFlags `json:"withdraw"`
}

type accessFlags struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

// newOrderMsg represents creating order payload
type newOrderMsg struct {
	CID    int64   `json:"cid"`
	Type   string  `json:"type"`
	Symbol string  `json:"symbol"`
	Amount float64 `json:"amount, string"`
	Price  float64 `json:"price, string"`
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
