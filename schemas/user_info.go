package schemas

// UserInfo - user info
type UserInfo struct {
	Access   Access
	Balances map[string]Balance
	Prices   map[string]float64
	TradesCount,
	OrdersCount int32
}

// Access - API keys access level
type Access struct {
	Read     bool
	Trade    bool
	Deposit  bool
	Withdraw bool
}

// Balance - balance struct
type Balance struct {
	Coin      string
	Available float64
	InOrders  float64
	Total     float64
}
