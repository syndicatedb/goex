package tidex

import "xproto/shared/schemas"

// SymbolsProvider - order book provider
type SymbolsProvider struct {
}

// NewSymbolsProvider - SymbolsProvider constructor
func NewSymbolsProvider() *SymbolsProvider {
	return &SymbolsProvider{}
}

// Get - getting all symbols from Exchange
func (sp *SymbolsProvider) Get() (symbols []schemas.Symbol, err error) {
	return
}
