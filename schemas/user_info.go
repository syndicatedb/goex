package schemas

type UserInfo struct {
	Access   Access
	Balances map[string]Balance
	TradesCount,
	OrdersCount int32
}

type Access struct {
	Read     bool
	Trade    bool
	Deposit  bool
	Withdraw bool
}

type Balance struct {
	Available float64
	InOrders  float64
	Total     float64
}
