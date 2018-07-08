package clients

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/syndicatedb/goproxy/proxy"
)

const (
	methodGET  = "GET"
	methodPOST = "POST"
)

// HTTP - http mapper/helper
type HTTP struct {
	proxy *proxy.Client
}

// NewHTTP - HTTP mapper constructor
func NewHTTP(proxy *proxy.Client) *HTTP {
	return &HTTP{
		proxy: proxy,
	}
}

// Params - map key=value for params and payload
type Params map[string]string

// Get - http GET request
func (client *HTTP) Get(url string, params Params) (b []byte, err error) {
	return client.Request(methodGET, url, params, nil)
}

// Request - custom HTTP request
func (client *HTTP) Request(method, endpoint string, params, payload Params) (b []byte, err error) {
	var formData string
	rawurl := endpoint
	if method == methodGET {
		var URL *url.URL
		URL, err = url.Parse(rawurl)
		if err != nil {
			return
		}
		q := URL.Query()
		for key, value := range params {
			q.Set(key, value)
		}
		formData = q.Encode()
		URL.RawQuery = formData
		rawurl = URL.String()
	} else {
		formValues := url.Values{}
		for key, value := range payload {
			formValues.Set(key, value)
		}
		formData = formValues.Encode()
	}
	// fmt.Println("formData: ", rawurl, formData)
	req, err := http.NewRequest(method, rawurl, strings.NewReader(formData))
	if err != nil {
		return
	}

	if method == "POST" || method == "PUT" {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	}
	req.Header.Add("Accept", "application/json")

	resp, err := client.proxy.Do(req)
	if err != nil {
		fmt.Println("Response: ", resp)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Println(resp.Status)
	}
	return body, nil
}
