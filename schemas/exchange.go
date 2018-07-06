package schemas

type Exchange struct {
	credentials       Credentials
	OrderBookProvider OrderBookProvider
}

func NewExchange(apiKey, apiSecret string) Exchange {
	return Exchange{
		credentials: Credentials{
			APIKey:    apiKey,
			APISecret: apiSecret,
		},
	}
}

type Credentials struct {
	APIKey    string
	APISecret string
}
