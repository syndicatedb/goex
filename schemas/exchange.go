package schemas

// Credentials - struct to store credentials for private requests
type Credentials struct {
	APIKey    string
	APISecret string
}

// Result - sending data with channels
type Result struct {
	DataType string
	Error    error
	Data     interface{}
}
