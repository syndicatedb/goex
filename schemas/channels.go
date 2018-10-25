package schemas

// ResultChannel - sending data with channels
type ResultChannel struct {
	DataType string
	Error    error
	Data     interface{}
}

// UserInfoChannel - for User info subscription
type UserInfoChannel struct {
	Data     UserInfo
	DataType string
	Error    error
}

// UserOrdersChannel - for User orders subscription
type UserOrdersChannel struct {
	Data     []Order
	DataType string
	Error    error
}

// UserTradesChannel - for User trades subscription
type UserTradesChannel struct {
	Data     []Trade
	DataType string
	Error    error
}
