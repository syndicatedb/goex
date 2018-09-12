package binance

import (
	httpclient "github.com/syndicatedb/goex/internal/http"

	"encoding/json"
	"log"
)

const url = "https://api.binance.com/api/v1/userDataStream"

type response struct {
	ListenKey string `json:"listenKey"`
}

func (trading *TradingProvider) CreateListenkey(apiKey string) (string, error) {
	var resp response
	params := httpclient.Params()

	b, err := trading.httpClient.Post(url, params, httpclient.KeyValue{}, true)
	if err != nil {
		log.Println("Error sending request", err)
		return "", err
	}

	err = json.Unmarshal(b, &resp)
	if err != nil {
		return "", err
	}
	return resp.ListenKey, nil
}

func (trading *TradingProvider) Ping() error {
	params := httpclient.Params()
	params.Set("listenKey", trading.listenKey)

	_, err := trading.httpClient.Request("PUT", url, params, httpclient.KeyValue{}, true)
	if err != nil {
		return err
	}
	return nil
}

func (trading *TradingProvider) Delete() error {
	params := httpclient.Params()
	params.Set("listenKey", trading.listenKey)

	_, err := trading.httpClient.Request("DELETE", url, params, httpclient.KeyValue{}, true)
	if err != nil {
		return err
	}
	return nil
}
