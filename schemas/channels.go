package schemas

// ResultChannel - sending data with channels
type ResultChannel struct {
	DataType string
	Error    error
	Data     interface{}
}

// UserInfoChannel - for User info subscription
type UserInfoChannel struct {
	Data  UserInfo
	Error error
}

// UserOrdersChannel - for User orders subscription
type UserOrdersChannel struct {
	Data  []Order
	Error error
}

// UserTradesChannel - for User trades subscription
type UserTradesChannel struct {
	Data  []Trade
	Error error
}
