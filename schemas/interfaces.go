package schemas

type SymbolProvider interface {
	Get()
	Subscribe()
}

type OrderBookProvider interface {
	Get()
	Subscribe()
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
