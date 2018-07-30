package websocket

import (
	"encoding/json"
	"log"
	"net/http"
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
}

/*
NewClient - Websocket client constructor
*/
func NewClient(url string, proxy proxy.Provider) *Client {
	return &Client{
		config:           Config{URL: url},
		keepalive:        true,
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
	var resp *http.Response
	// proxyClient := c.proxyProvider.NewClient("")
	dialer := websocket.Dialer{
	// Proxy: http.ProxyURL(),
	}
	c.conn, resp, err = dialer.Dial(c.getAddressURL(), nil)
	// c.conn, resp, err = websocket.DefaultDialer.Dial(c.getAddressURL(), nil)
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
		log.Printf("c.conn: %+v\n", c.conn)
	}
	c.channel = ch
	c.errorChannel = ech
	c.done = make(chan struct{})

	go func() {
		defer close(c.done)
		for {
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				c.errorChannel <- NewReadError(err)
				continue
			}

			if c.channel != nil {
				c.channel <- message
			}
		}
	}()
	if c.keepalive {
		go c.keepAlive()
	}
}

// Write - writing to websocket
func (c *Client) Write(data interface{}) (err error) {
	if c.conn == nil {
		log.Printf("c.conn: %+v\n", c.conn)
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
