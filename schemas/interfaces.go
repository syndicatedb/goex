package schemas

import (
	"time"
)

// SymbolProvider - provides symbol methods
type SymbolProvider interface {
	Get() (symbols []Symbol, err error)
	Subscribe(time.Duration) chan Result
}

// OrdersProvider - provides access to Order book
type OrdersProvider interface {
	SetSymbols(symbols []Symbol) OrdersProvider
	GetOrderBook(symbol Symbol) (book OrderBook, err error)
	Subscribe(symbol Symbol, d time.Duration) chan Result
	SubscribeAll(d time.Duration) chan Result
}

// QuotesProvider - provides quotes/ticker
type QuotesProvider interface {
	SetSymbols(symbols []Symbol) QuotesProvider
	Get(symbol Symbol) (q Quote, err error)
	Subscribe(symbol Symbol, d time.Duration) chan Result
	SubscribeAll(d time.Duration) chan Result
}

// TradesProvider - provides public trades
type TradesProvider interface {
	SetSymbols(symbols []Symbol) TradesProvider
	Get(symbol Symbol) (t []Trade, err error)
	Subscribe(symbol Symbol, d time.Duration) chan Result
	SubscribeAll(d time.Duration) chan Result
}

// type OHLCVProvider interface {
// 	Get()
// 	Subscribe()
// }

// type UserProvider interface {
// 	Get()
// 	Balance()
// }

// type UserBalanceProvider interface {
// 	Get()
// 	Subscribe()
// }

// type UserOrdersProvider interface {
// 	Get()
// 	Subscribe()
// 	Create()
// 	Cancel()
// 	CancelAll()
// }

// type UserTradesProvider interface {
// 	Get()
// 	Subscribe()
// }
