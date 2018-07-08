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

type TradeProvider interface {
	Get()
	Subscribe()
}

type QuoteProvider interface {
	Get()
	Subscribe()
}

type OHLCVProvider interface {
	Get()
	Subscribe()
}

type UserProvider interface {
	Get()
	Balance()
}

type UserBalanceProvider interface {
	Get()
	Subscribe()
}

type UserOrdersProvider interface {
	Get()
	Subscribe()
	Create()
	Cancel()
	CancelAll()
}

type UserTradesProvider interface {
	Get()
	Subscribe()
}
