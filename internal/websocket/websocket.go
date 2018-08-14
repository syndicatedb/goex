package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/syndicatedb/goproxy/proxy"
)

/*
Config - websocket config
*/
type Config struct {
	URL string
}

// Client - Websocket client
type Client struct {
	config           Config
	pingMessage      string
	keepalive        bool
	channel          chan []byte
	errorChannel     chan error
	conn             *websocket.Conn
	keepaliveTimeout time.Duration
	done             chan struct{}

	proxyProvider proxy.Provider

	mu sync.RWMutex
}

/*
NewClient - Websocket client constructor
*/
func NewClient(url string, proxy proxy.Provider) *Client {
	return &Client{
		config:           Config{URL: url},
		keepalive:        false,
		keepaliveTimeout: time.Minute,
		proxyProvider:    proxy,
	}
}

/*
UsePingMessage - setting ping message string that will be sent to keep alive
*/
func (c *Client) UsePingMessage(pm string) *Client {
	c.pingMessage = pm
	return c
}

/*
ChangeKeepAlive - setting keepalive flag
*/
func (c *Client) ChangeKeepAlive(f bool) *Client {
	c.keepalive = f
	return c
}

// Connect - connecting to Websocket server
func (c *Client) Connect() (err error) {
	log.Println("websocket connecting")
	var dialer websocket.Dialer
	var resp *http.Response
	ip := c.proxyProvider.IP()
	if len(ip) > 0 {
		proxyURL, err := url.Parse(c.proxyProvider.IP())
		if err != nil {
			log.Println("Error while connecting through proxy", err)
		}
		dialer = websocket.Dialer{
			Proxy:            http.ProxyURL(proxyURL),
			HandshakeTimeout: 30 * time.Second,
		}
	} else {
		dialer = websocket.Dialer{}
	}
	c.conn, resp, err = dialer.Dial(c.getAddressURL(), nil)
	if err != nil {
		log.Println("ws connection error: ", err)
		log.Println("ws connection error response: ", resp)
		return NewConnectionError(err)
	}
	log.Println("websocket connected")
	return
}

// Exit - graceful exit and closing connection
func (c *Client) Exit() (err error) {
	err = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return NewCloseConnectionError(err)
	}
	c.conn.Close()
	return
}

// Listen - starting to receive messages
func (c *Client) Listen(ch chan []byte, ech chan error) {
	if c.conn == nil {
		err := fmt.Errorf("WS connection is nil")
		c.errorChannel <- NewReadError(err)
	}
	c.channel = ch
	c.errorChannel = ech
	c.done = make(chan struct{})

	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Println("Recovered after error: ", err)
			}
		}()
		defer close(c.done)
		for {
			var data interface{}
			// _, message, err := c.conn.ReadMessage()
			err := c.conn.ReadJSON(&data)
			if err != nil {
				log.Println("Err", err)
				c.errorChannel <- NewReadError(err)
				return
			}

			message, err := json.Marshal(data)
			if err != nil {
				log.Println("Err", err)
				c.errorChannel <- NewReadError(err)
				return
			}

			if c.channel == nil {
				c.errorChannel <- NewChannelNilError()
				return
			}
			c.channel <- message
		}
	}()
	if c.keepalive {
		go c.keepAlive()
	}
}

// Write - writing to websocket
func (c *Client) Write(data interface{}) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		err = fmt.Errorf("WS connection is nil: %v", err)
		return
	}
	var b []byte
	if b, err = json.Marshal(data); err != nil {
		return
	}
	return c.conn.WriteMessage(websocket.TextMessage, b)
}

func (c *Client) keepAlive() {
	ticker := time.NewTicker(c.keepaliveTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			// fmt.Println("Sending ping: ", t)
			if err := c.conn.WriteMessage(websocket.TextMessage, []byte(c.pingMessage)); err != nil {
				c.errorChannel <- NewKeepaliveError(err)
			}
		}
	}
}

func (c *Client) getAddressURL() string {
	return c.config.URL
	// return fmt.Sprintf("%s://%s:%d", c.config.Protocol, c.config.Host, c.config.Port)
}
