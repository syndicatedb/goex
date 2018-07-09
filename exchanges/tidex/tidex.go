package tidex

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"net/url"
	"strings"
	"time"

	"github.com/syndicatedb/goex/clients"
	"github.com/syndicatedb/goex/schemas"
)

const (
	// URL - API endpoint
	apiSymbols   = "https://api.tidex.com/api/3/info"
	apiOrderBook = "https://api.tidex.com/api/3/depth/"
	apiQuotes    = "https://api.tidex.com/api/3/ticker/"
	apiTrades    = "https://api.tidex.com/api/3/trades/"
	apiUserInfo  = "https://api.tidex.com/tapi"
)

var (
	// SubscriptionInterval - default subscription interval
	SubscriptionInterval  = 1 * time.Second
	orderBookSymbolsLimit = 20
	quotesSymbolsLimit    = 10
	exchangeName          = ""
)

/*
Tidex - exchange struct
*/
type Tidex struct {
	schemas.Exchange
}

// New - Tidex constructor. APIKey and APISecret is mandatory, but could be empty
func New(opts schemas.Options) *Tidex {
	exchangeName = opts.Name
	proxyProvider := opts.ProxyProvider
	if proxyProvider == nil {
		proxyProvider = clients.NewNoProxy()
	}
	return &Tidex{
		Exchange: schemas.Exchange{
			Credentials:    opts.Credentials,
			ProxyProvider:  proxyProvider,
			SymbolProvider: NewSymbolsProvider(proxyProvider),
			OrdersProvider: NewOrdersProvider(proxyProvider),
			QuotesProvider: NewQuotesProvider(proxyProvider),
			TradesProvider: NewTradesProvider(proxyProvider),
			UserProvider:   NewUserProvider(opts.Credentials, proxyProvider),
		},
	}
}

func parseSymbol(s string) (name, coin, baseCoin string) {
	sa := strings.Split(s, "_")
	coin = strings.ToUpper(sa[0])
	baseCoin = strings.ToUpper(sa[1])
	name = coin + "-" + baseCoin
	return
}

func signRequest(apiSecret string, payload map[string]string) string {
	formValues := url.Values{}
	for key, value := range payload {
		formValues.Set(key, value)
	}
	formData := formValues.Encode()

	sig := hmac.New(sha512.New, []byte(apiSecret))
	sig.Write([]byte(formData))

	return hex.EncodeToString(sig.Sum(nil))
}
