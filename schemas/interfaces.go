package schemas

import (
	"net/http"
	"time"
)

// Signer - signing request for Private API
type Signer func(string, string, *http.Request) *http.Request

// SymbolProvider - provides symbol methods
type SymbolProvider interface {
	Get() (symbols []Symbol, err error)
	Subscribe(time.Duration) chan ResultChannel
}

// OrdersProvider - provides access to Order book
type OrdersProvider interface {
	SetSymbols(symbols []Symbol) OrdersProvider
	GetOrderBook(symbol Symbol) (book OrderBook, err error)
	subscriber
}

// QuotesProvider - provides quotes/ticker
type QuotesProvider interface {
	SetSymbols(symbols []Symbol) QuotesProvider
	Get(symbol Symbol) (q Quote, err error)
	subscriber
}

// TradesProvider - provides public trades
type TradesProvider interface {
	SetSymbols(symbols []Symbol) TradesProvider
	Get(symbol Symbol) (t []Trade, err error)
	subscriber
}

// subscriber - provides public trades
type subscriber interface {
	Subscribe(symbol Symbol, d time.Duration) chan ResultChannel
	SubscribeAll(d time.Duration) chan ResultChannel
}

// type OHLCVProvider interface {
// 	Get()
// 	Subscribe()
// }

// TradingProvider - provides API to trade
type TradingProvider interface {
	Info() (UserInfo, error)
	Orders(symbols []Symbol) ([]Order, error)
	Trades(TradeHistoryOptions) ([]Trade, error)

	Subscribe(time.Duration) (chan UserInfoChannel, chan UserOrdersChannel, chan UserTradesChannel)
	Create(order Order) (result Order, err error)
	Cancel(order Order) (err error)
	CancelAll() (err error)
}
