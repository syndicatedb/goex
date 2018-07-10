package schemas

// UserInfoChannel - for User info subscription
type UserInfoChannel struct {
	Data  UserInfo
	Error error
}
