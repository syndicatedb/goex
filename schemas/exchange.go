package schemas

type Credentials struct {
	APIKey    string
	APISecret string
}

// Result - sending data with channels
type Result struct {
	Error error
	Data  interface{}
}
