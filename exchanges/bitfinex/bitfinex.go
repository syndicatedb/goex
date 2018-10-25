package bitfinex

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/syndicatedb/goex/internal/proxy"
	"github.com/syndicatedb/goex/schemas"
)

const (
	exchangeName   = "bitfinex"
	apiSymbols     = "https://api.bitfinex.com/v1/symbols_details"
	apiOrderBook   = "https://api.bitfinex.com/v2/book"
	apiTrades      = "https://api.bitfinex.com/v2/trades"
	apiQuotes      = "https://api.bitfinex.com/v2/ticker"
	apiCandles     = "https://api.bitfinex.com/v2/candles"
	apiAccess      = "https://api.bitfinex.com/v1/key_info"
	apiMyTrades    = "https://api.bitfinex.com/v1/mytrades"
	apiNewOrder    = "https://api.bitfinex.com/v1/order/new"
	apiCancelOrder = "https://api.bitfinex.com/v1/order/cancel"
	apiCancelAll   = "https://api.bitfinex.com/v1/order/cancel/all"

	apiURL = "https://api.bitfinex.com"
	wsURL  = "wss://api.bitfinex.com/ws/2"
)

const (
	subscriptionInterval  = 1 * time.Second
	snapshotInterval      = 5 * time.Minute
	orderBookSymbolsLimit = 100
	tradesSymbolsLimit    = 10
	quotesSymbolsLimit    = 10
)

const (
	dataTypeSnapshot = "s"
	dataTypeUpdate   = "u"
)

// Bitfinex - bitfinex exchange structure
type Bitfinex struct {
	schemas.Exchange
}

// New - bitfinex exchange constructor
func New(opts schemas.Options) *Bitfinex {
	proxyProvider := opts.ProxyProvider
	if proxyProvider == nil {
		proxyProvider = proxy.NewNoProxy()
	}

	opts.Credentials.Sign = signV1
	return &Bitfinex{
		Exchange: schemas.Exchange{
			Credentials:   opts.Credentials,
			ProxyProvider: proxyProvider,
			Symbol:        NewSymbolsProvider(proxyProvider),
			Orders:        NewOrdersProvider(proxyProvider),
			Trades:        NewTradesProvider(proxyProvider),
			Quotes:        NewQuotesProvider(proxyProvider),
			Candles:       NewCandlesProvider(proxyProvider),
			Trading:       NewTradingProvider(opts.Credentials, proxyProvider),
		},
	}
}

func parseSymbol(smb string) (name, basecoin, quoteCoin string) {
	if strings.Index(smb, "t") == 0 {
		smb = strings.Replace(smb, "t", "", 1)
	}
	s := strings.ToUpper(smb)
	baseSymbols := []string{"USD", "EUR", "GBP", "JPY", "BTC", "ETH", "EOS"}

	for _, symb := range baseSymbols {
		if strings.Contains(s, symb) {
			if strings.LastIndex(s, symb)+2 == len(s)-1 {
				quoteCoin = strings.ToUpper(symb)
				basecoin = strings.ToUpper((strings.Replace(s, symb, "", -1)))
			}
		}

	}
	name = basecoin + "-" + quoteCoin
	return
}

func unparseSymbol(symbol string) string {
	sa := strings.Split(symbol, "-")

	return sa[0] + sa[1]
}

func signV1(key, secret string, req *http.Request) *http.Request {
	b, _ := req.GetBody()
	body, err := ioutil.ReadAll(b)
	if err != nil {
		return req
	}

	payloadEnc := base64.StdEncoding.EncodeToString(body)
	sig := createSignature384(payloadEnc, secret)
	req.Header.Add("X-BFX-APIKEY", key)
	req.Header.Add("X-BFX-PAYLOAD", payloadEnc)
	req.Header.Add("X-BFX-SIGNATURE", sig)

	return req
}

func signV2(key, secret, path string, req *http.Request) *http.Request {
	b, _ := req.GetBody()
	body, err := ioutil.ReadAll(b)
	if err != nil {
		return req
	}

	// nonce := fmt.Sprintf("%v", time.Now().Unix()*10000)
	nonce := fmt.Sprintf("%v", time.Now().UnixNano()/1000)
	str := "/api" + path + nonce + string(body)
	log.Println("STR", str)
	sig := createSignature384(str, secret)
	req.Header.Add("bfx-nonce", nonce)
	req.Header.Add("bfx-apikey", key)
	req.Header.Add("bfx-signature", sig)
	req.Header.Add("Content-Type", "application/json")

	return req
}

func createSignature384(msg, secret string) string {
	hash := hmac.New(sha512.New384, []byte(secret))
	hash.Write([]byte(msg))
	return hex.EncodeToString(hash.Sum(nil))
}
