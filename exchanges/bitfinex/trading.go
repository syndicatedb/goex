package bitfinex

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/syndicatedb/goex/internal/http"
	"github.com/syndicatedb/goex/internal/websocket"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

const (
	errConnecting = "Error connecting to bitfinex WS: %v"
	errAuth       = "Bitfinex auth error: %v"
	errOnWs       = "Error from websocket client: %v"
)

// TradingProvider represents bitfinex trading provider structure
type TradingProvider struct {
	credentials schemas.Credentials
	wsClient    *websocket.Client
	httpClient  *httpclient.Client
	proxyClient proxy.Client

	bus tradingBus
}

type tradingBus struct {
	uic chan schemas.UserInfoChannel
	uoc chan schemas.UserOrdersChannel
	utc chan schemas.UserTradesChannel
}

// NewTradingProvider constructing bitfinex trading provider
func NewTradingProvider(creds schemas.Credentials, proxy proxy.Provider) *TradingProvider {
	proxyClient := proxy.NewClient(exchangeName)
	wsClient := websocket.NewClient(wsURL, proxy)

	return &TradingProvider{
		credentials: creds,
		wsClient:    wsClient,
		httpClient:  httpclient.NewSigned(creds, proxyClient),
		proxyClient: proxyClient,
		bus: tradingBus{
			uic: make(chan schemas.UserInfoChannel),
			uoc: make(chan schemas.UserOrdersChannel),
			utc: make(chan schemas.UserTradesChannel),
		},
	}
}

// Subscribe subscribing to accounts updates for balances, orders, trades
func (trading *TradingProvider) Subscribe(interval time.Duration) (chan schemas.UserInfoChannel, chan schemas.UserOrdersChannel, chan schemas.UserTradesChannel) {
	dch := make(chan []byte, 100)
	ech := make(chan error, 100)

	if err := trading.wsClient.Connect(); err != nil {
		err = fmt.Errorf(errConnecting, err)
		trading.publishErr(err)
	}
	trading.wsClient.ChangeKeepAlive(false)
	trading.wsClient.Listen(dch, ech)

	go func() {
		for {
			select {
			case msg := <-dch:
				log.Println("Incoming message: ", string(msg))
				trading.handleMessages(msg)
			case err := <-ech:
				log.Printf(errOnWs, err)
				err = fmt.Errorf(errOnWs, err)
				trading.publishErr(err)
			}
		}
	}()

	if err := trading.auth(); err != nil {
		log.Printf(errAuth, err)
		err = fmt.Errorf(errAuth, err)
		trading.publishErr(err)
	}

	return trading.bus.uic, trading.bus.uoc, trading.bus.utc
}

// Info stub method
func (trading *TradingProvider) Info() (ui schemas.UserInfo, err error) {
	return
}

// Orders stub method
func (trading *TradingProvider) Orders(symbols []schemas.Symbol) (orders []schemas.Order, err error) {
	return
}

// Trades stub method
func (trading *TradingProvider) Trades(opts schemas.FilterOptions) (trades []schemas.Trade, p schemas.Paging, err error) {
	return
}

// ImportTrades stub method
func (trading *TradingProvider) ImportTrades(opts schemas.FilterOptions) chan schemas.UserTradesChannel {
	ch := make(chan schemas.UserTradesChannel)
	return ch
}

// Create stub method
func (trading *TradingProvider) Create(order schemas.Order) (result schemas.Order, err error) {
	return
}

// Cancel stub method
func (trading *TradingProvider) Cancel(order schemas.Order) (err error) {
	return
}

// CancelAll stub method
func (trading *TradingProvider) CancelAll() (err error) {
	return
}

func (trading *TradingProvider) auth() error {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)[:13]

	payload := "AUTH" + nonce
	signature := createSignature384(payload, trading.credentials.APISecret)

	msg := authMsg{
		Event:       "auth",
		APIKey:      trading.credentials.APIKey,
		AuthSig:     signature,
		AuthPayload: payload,
		AuthNonce:   nonce,
	}

	return trading.wsClient.Write(msg)
}

func (trading *TradingProvider) handleMessages(data []byte) {
	var msg interface{}

	err := json.Unmarshal(data, &msg)
	if err != nil {
		log.Println("Error unmarshalling message: ", err)
		return
	}

	if eventMsg, ok := msg.(map[string]interface{}); ok {
		if eventMsg["event"] != nil {
			if eventMsg["event"] == "auth" {
				err = trading.checkAuthMessage(eventMsg)
				if err != nil {
					trading.publishErr(err)
					return
				}
			} else {
				err := trading.handleEvents(eventMsg)
				if err != nil {
					trading.publishErr(err)
					return
				}
			}
		}
	}

	if updateMsg, ok := msg.([]interface{}); ok {
		trading.handleUpdates(updateMsg)
	}
}

func (trading *TradingProvider) handleUpdates(msg []interface{}) {
	updType := msg[1]

	if updType == "ws" {
		b := trading.mapBalance(msg[2].([]interface{}))
		access, err := trading.getAccessInfo()
		if err != nil {
			trading.publishErr(err)
			return
		}

		log.Println("Access data", access)
		trading.bus.uic <- schemas.UserInfoChannel{
			Data: schemas.UserInfo{
				Access:   access,
				Balances: b,
			},
		}
	}
	if updType == "wu" {
		wslice := []interface{}{msg[2]}
		b := trading.mapBalance(wslice)
		access, err := trading.getAccessInfo()
		if err != nil {
			trading.publishErr(err)
			return
		}
		log.Println("Access data", access)

		trading.bus.uic <- schemas.UserInfoChannel{
			Data: schemas.UserInfo{
				Access:   access,
				Balances: b,
			},
		}
	}
	if updType == "os" {
		m := trading.mapOrders(msg[2].([]interface{}))
		trading.bus.uoc <- m
	}
	if updType == "on" || updType == "ou" || updType == "oc" {
		wslice := []interface{}{msg[2]}
		m := trading.mapOrders(wslice)
		trading.bus.uoc <- m
	}
	if updType == "tu" {
		m := trading.mapTrades(msg[2].([]interface{}))
		trading.bus.utc <- m
	}
}

func (trading *TradingProvider) handleEvents(msg map[string]interface{}) error {
	if msg["event"] == "error" {
		log.Println("WS error: ", msg)
		return errors.New("WS error: " + msg["msg"].(string))
	}
	if msg["event"] == "info" {
		log.Println("Info message: ", msg)
	}

	return nil
}

func (trading *TradingProvider) checkAuthMessage(msg map[string]interface{}) error {
	if msg["status"] == "OK" {
		log.Println("WS auth is ok")
		return nil
	}

	return errors.New("WS subscription not authorized")
}

func (trading *TradingProvider) getAccessInfo() (access schemas.Access, err error) {
	log.Println("Getting access data")
	var resp accessResponse

	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)[:13]
	req, err := http.NewRequest("POST", apiAccess, nil)
	if err != nil {
		return
	}
	payload := map[string]interface{}{
		"request": "/v1/key_info",
		"nonce":   nonce,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}
	payloadEnc := base64.StdEncoding.EncodeToString(payloadBytes)
	sig := createSignature384(payloadEnc, trading.credentials.APISecret)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-BFX-APIKEY", trading.credentials.APIKey)
	req.Header.Add("X-BFX-PAYLOAD", payloadEnc)
	req.Header.Add("X-BFX-SIGNATURE", sig)

	r, err := trading.proxyClient.Do(req)
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		return
	}

	access.Read = resp.Account.Read
	access.Trade = resp.Orders.Write
	access.Withdraw = resp.Withdraw.Write

	return
}

func (trading *TradingProvider) publishErr(err error) {
	go func() {
		trading.bus.uic <- schemas.UserInfoChannel{
			Error: err,
		}
	}()
}

func (trading *TradingProvider) mapBalance(msg []interface{}) map[string]schemas.Balance {
	sb := make(map[string]schemas.Balance)

	for i := range msg {
		if wal, ok := msg[i].([]interface{}); ok {
			b := schemas.Balance{
				Coin:  wal[1].(string),
				Total: wal[2].(float64),
			}
			if aval, ok := wal[4].(float64); ok {
				b.Available = aval
				b.InOrders = b.Total - b.Available
			}

			sb[wal[1].(string)] = b
		}
	}

	return sb
}

func (trading *TradingProvider) mapOrders(msg []interface{}) schemas.UserOrdersChannel {
	var orders []schemas.Order

	for i := range msg {
		if ord, ok := msg[i].([]interface{}); ok {
			var side string
			// var status string

			symbol, _, _ := parseSymbol(ord[3].(string))

			if ord[6].(float64) > 0 {
				side = schemas.TypeBuy
			} else {
				side = schemas.TypeSell
			}

			// if ord[10] == "EXECUTED" {
			// 	status = schemas.StatusTrade
			// } else if ord[10] == "ACTIVE" {
			// 	status = schemas.StatusNew
			// } else if ord[10] == "CANCELED" {
			// 	status = schemas.StatusCancelled
			// } else {
			// 	status = ord[10].(string)
			// }

			order := schemas.Order{
				ID:     strconv.FormatFloat(ord[0].(float64), 'f', -1, 64),
				Symbol: symbol,
				Type:   side,
				// Status: status,
				Price:     ord[11].(float64),
				Amount:    ord[7].(float64),
				CreatedAt: ord[4].(int64),
			}

			orders = append(orders, order)
		}
	}

	return schemas.UserOrdersChannel{
		Data: orders,
	}
}

func (trading *TradingProvider) mapTrades(msg []interface{}) schemas.UserTradesChannel {
	var trades []schemas.Trade

	for i := range msg {
		if trd, ok := msg[i].([]interface{}); ok {
			var side string

			symbol, _, _ := parseSymbol(trd[1].(string))

			if trd[4].(float64) > 0 {
				side = schemas.TypeBuy
			} else {
				side = schemas.TypeSell
			}

			trade := schemas.Trade{
				ID:        strconv.FormatInt(trd[0].(int64), 10),
				OrderID:   strconv.FormatInt(trd[3].(int64), 10),
				Symbol:    symbol,
				Type:      side,
				Timestamp: trd[2].(int64),
				Amount:    trd[4].(float64),
				Price:     trd[5].(float64),
				Fee:       trd[9].(float64),
			}

			trades = append(trades, trade)
		}
	}

	return schemas.UserTradesChannel{
		Data: trades,
	}
}
