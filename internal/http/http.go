package httpclient

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

const (
	methodGET  = "GET"
	methodPOST = "POST"
)

// Client - http mapper/helper
type Client struct {
	proxy       proxy.Client
	credentials schemas.Credentials
	Headers     KeyValue
}

// NewSigned - HTTP mapper constructor
func NewSigned(credentials schemas.Credentials, proxy proxy.Client) *Client {
	return &Client{
		proxy:       proxy,
		credentials: credentials,
		Headers:     Headers(),
	}
}

// New - HTTP mapper constructor
func New(proxy proxy.Client) *Client {
	return &Client{
		proxy:   proxy,
		Headers: Headers(),
	}
}

type KeyValue struct {
	data map[string]string
}

func (p *KeyValue) Set(key, value string) *KeyValue {
	p.data[key] = value
	return p
}

func (p *KeyValue) Map() map[string]string {
	return p.data
}

// Params - map key=value for params and payload
func Params() KeyValue {
	return KeyValue{
		data: make(map[string]string),
	}
}

// Headers - map key=value to set HTTP headers
func Headers() KeyValue {
	return Params()
}

// Get - http GET request
func (client *Client) Get(url string, params KeyValue, isSigned bool) (b []byte, err error) {
	return client.Request(methodGET, url, params, KeyValue{}, isSigned)
}

// Post - http GET request
func (client *Client) Post(url string, params, payload KeyValue, isSigned bool) (b []byte, err error) {
	return client.Request(methodPOST, url, params, payload, isSigned)
}

// Request - custom HTTP request
func (client *Client) Request(method, endpoint string, params, payload KeyValue, isSigned bool) (b []byte, err error) {
	var formData string
	rawurl := endpoint
	if method == methodGET {
		var URL *url.URL
		URL, err = url.Parse(rawurl)
		if err != nil {
			return
		}
		q := URL.Query()
		for key, value := range params.data {
			q.Set(key, value)
		}
		formData = q.Encode()
		URL.RawQuery = formData
		rawurl = URL.String()
	} else {
		formValues := url.Values{}
		for key, value := range payload.data {
			formValues.Set(key, value)
		}
		formData = formValues.Encode()
	}

	req, err := http.NewRequest(method, rawurl, strings.NewReader(formData))
	if err != nil {
		return
	}

	if method == "POST" || method == "PUT" {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	}
	req.Header.Add("Accept", "application/json,text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.99 Safari/537.36")

	if isSigned {
		req = client.sign(req)
	}

	if len(client.Headers.data) > 0 {
		for key, v := range client.Headers.data {
			req.Header.Add(key, v)
		}
	}

	resp, err := client.proxy.Do(req)
	if err != nil {
		fmt.Println("Error: ", err)
		fmt.Printf("Response: %+v\n\n", resp)
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading body:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println(resp.Status)
	}
	return body, nil
}

func (client *Client) sign(req *http.Request) *http.Request {
	key := client.credentials.APIKey
	secret := client.credentials.APISecret
	return client.credentials.Sign(key, secret, req)
}
