package httpclient

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
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

var headers = []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/42.0.2311.135 Safari/537.36 Edge/12.246",
	"Mozilla/5.0 (Linux; Android 5.0.2; SAMSUNG SM-T550 Build/LRX22G) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/3.3 Chrome/38.0.2125.102 Safari/537.36",
	"Mozilla/5.0 (Nintendo WiiU) AppleWebKit/536.30 (KHTML, like Gecko) NX/3.0.4.2.12 NintendoBrowser/4.3.1.11264.US",
	"Mozilla/5.0 (PlayStation 4 3.11) AppleWebKit/537.73 (KHTML, like Gecko)",
	"Mozilla/5.0 (Linux; Android 5.0.2; LG-V410/V41020c Build/LRX22G) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/34.0.1847.118 Safari/537.36",
	"Mozilla/5.0 (Linux; Android 7.0; SM-G892A Build/NRD90M; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/60.0.3112.107 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 7.0; Pixel C Build/NRD90M; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/52.0.2743.98 Safari/537.36",
	"Mozilla/5.0 (Linux; Android 6.0.1; SHIELD Tablet K1 Build/MRA58K; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/55.0.2883.91 Safari/537.36",
	"Mozilla/5.0 (X11; CrOS x86_64 8172.45.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.64 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_2) AppleWebKit/601.3.9 (KHTML, like Gecko) Version/9.0.2 Safari/601.3.9",
	"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.111 Safari/537.36",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:15.0) Gecko/20100101 Firefox/15.0.1",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; XBOX_ONE_ED) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.79 Safari/537.36 Edge/14.14393",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.99 Safari/537.36",
}

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
	// log.Println("ENDPOINT", rawurl)
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
	req.Header.Add("User-Agent", headers[rand.Intn(14)])

	if isSigned {
		req = client.sign(req)
	}
	log.Println("req.URL: ", req.URL)
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
		log.Println("Data:", string(body), "Error:", err)
		// log.Println("Resp status is:", resp.Status)
		err = fmt.Errorf("Status code is: %v", resp.StatusCode)
		return
	}
	return body, nil
}

func (client *Client) sign(req *http.Request) *http.Request {
	key := client.credentials.APIKey
	secret := client.credentials.APISecret
	return client.credentials.Sign(key, secret, req)
}
