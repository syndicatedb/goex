package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

// ContentTypes for requests
const (
	ContentTypeJSON = "application/json"
	ContentTypeForm = "application/x-www-form-urlencoded;charset=utf-8"
)

// Client - http mapper/helper
type Client struct {
	proxy       proxy.Client
	credentials schemas.Credentials
	Headers     KeyValue
	ContentType string
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
	var body io.Reader
	body = nil

	rawurl := endpoint
	var URL *url.URL
	// log.Println("ENDPOINT", rawurl)
	if len(params.data) > 0 {
		URL, err = url.Parse(rawurl)
		if err != nil {
			return
		}
		q := URL.Query()
		for key, value := range params.data {
			q.Set(key, value)
		}
		URL.RawQuery = q.Encode()
		rawurl = URL.String()
	}

	if len(payload.data) > 0 {
		var err error

		URL, err = url.Parse(rawurl)
		if err != nil {
			return nil, err
		}
		q := URL.Query()
		for key, value := range payload.data {
			q.Set(key, value)
		}
		formData = q.Encode()
		URL.RawQuery = formData
		rawurl = URL.String()

		body = client.getBody(URL, payload)
	}

	req, err := http.NewRequest(method, rawurl, body)
	if err != nil {
		return
	}

	if method == "POST" || method == "PUT" {
		req.Header.Add("Content-Type", client.getContentType())
	}
	req.Header.Add("Accept", "application/json,text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")

	if isSigned {
		req = client.sign(req)
	}
	// log.Println("req.URL: ", req.URL)
	if len(client.Headers.data) > 0 {
		for key, v := range client.Headers.data {
			req.Header.Add(key, v)
		}
	}

	return client.Do(req)
}

func (client *Client) getBody(u *url.URL, payload KeyValue) io.Reader {
	if u == nil {
		return strings.NewReader("")
	}
	log.Printf("u: %+v\n", u.Query())
	if client.getContentType() == ContentTypeForm {
		return strings.NewReader(u.RawQuery)
	}
	log.Printf("u: %+v\n", u)
	data := payload.Map()
	b, err := json.Marshal(data)
	if err != nil {
		log.Println("err: ", err)
	}
	return bytes.NewBuffer(b)
}

func (client *Client) getContentType() string {
	if client.ContentType == "" {
		return ContentTypeForm
	}
	return client.ContentType
}

// Do making HTTP request, can be user for custom requests
func (client *Client) Do(req *http.Request) (b []byte, err error) {
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
		log.Println("Data:", string(body), "Error:", err)
		// log.Println("Resp status is:", resp.Status)
		err = fmt.Errorf("Status code is: %v", resp.StatusCode)
		return body, err
	}
	return body, nil
}

func (client *Client) sign(req *http.Request) *http.Request {
	key := client.credentials.APIKey
	secret := client.credentials.APISecret
	return client.credentials.Sign(key, secret, req)
}
