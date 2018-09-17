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
	Type            string  `json:"type"`
	Symbol          string  `json:"symbol"`
	Amount          float64 `json:"amount"`
	Price           float64 `json:"price"`
	IsHidden        bool    `json:"is_hidden"`
	IsPostOnly      bool    `json:"is_postonly"`
	UseAllAvailable int32   `json:"use_all_available"`
	OcoOrder        bool    `json:"ocoorder"`
	BuyPriceOco     float64 `json:"buy_price_oco"`
	SellPriceOco    float64 `json:"sell_price_oco"`
}

// newOrderResponse represtns newly created order response
type newOrderResponse struct {
	ID                int64  `json:"id"`
	OrderID           int64  `json:"order_id"`
	Symbol            string `json:"symbol"`
	Exchange          string `json:"exchange"`
	Price             string `json:"price"`
	AvgExecutionPrice string `json:"avg_execution_price"`
	Side              string `json:"side"`
	Type              string `json:"type"`
	Timestamp         string `json:"timestamp"`
	IsLive            bool   `json:"is_live"`
	IsCancelled       bool   `json:"is_cancelled"`
	IsHidden          bool   `json:"is_hidden"`
	WasForced         bool   `json:"was_forced"`
	OriginalAmount    string `json:"original_amount"`
	RemainigAmount    string `json:"remaining_amount"`
	ExecutedAmount    string `json:"executed_amount"`
}

// cancelAllResponse represents response model on cancelling all orders
type cancelAllResponse struct {
	Result string `json:"result"`
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
