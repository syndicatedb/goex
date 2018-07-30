package websocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	uuid "github.com/nu7hatch/gouuid"
)

// Connection - ws server upgraded connection
type Connection struct {
	ID          string
	conn        *websocket.Conn
	req         *http.Request
	dataChannel DataChannel
	subs        map[string]bool // array of subscribed channels
	subsChannel SubsChannel
	sync.RWMutex
}

// Message - struct in data channel
type Message struct {
	Type  string
	Conn  *Connection
	Data  []byte
	Error error
}

// NewConnection - ws connection constructor
func NewConnection(conn *websocket.Conn, req *http.Request, dc DataChannel, sc SubsChannel) *Connection {

	uid, _ := uuid.NewV4()

	return &Connection{
		ID:          uid.String(),
		conn:        conn,
		req:         req,
		dataChannel: dc,
		subsChannel: sc,
		subs:        make(map[string]bool),
	}
}

// Listen - listening for WS messages
func (c *Connection) Listen() (err error) {
	defer func() {
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.dataChannel <- Message{
				Type:  "error",
				Conn:  c,
				Data:  message,
				Error: err,
			}
			break
		}
		if c.dataChannel != nil {
			c.dataChannel <- Message{
				Type: "message",
				Conn: c,
				Data: message,
			}
		}
	}
	return
}

// Write - sending message to Websocket
func (c *Connection) Write(data interface{}) (err error) {
	var message []byte

	if err = c.check(); err != nil {
		return
	}
	if message, err = json.Marshal(data); err != nil {
		return
	}
	c.Lock()
	defer c.Unlock()
	err = c.conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		return err
	}
	return nil
}

// Subscribe - subscribing connection to channel
func (c *Connection) Subscribe(channel string) {
	c.Lock()
	c.subs[channel] = true
	c.Unlock()
	log.Println("c.subs: ", c.subs)
	c.subsChannel <- Subscription{
		isAdd:   true,
		channel: channel,
		conn:    c,
	}
}

// Unsubscribe - unsubscribing connection from channel
func (c *Connection) Unsubscribe(channel string) {
	c.Lock()
	if _, ok := c.subs[channel]; ok {
		delete(c.subs, channel)
	}
	c.Unlock()
	log.Println("c.subs: ", c.subs)
	c.subsChannel <- Subscription{
		isAdd:   false,
		channel: channel,
		conn:    c,
	}
}

func (c *Connection) check() error {
	if c.conn == nil {
		fmt.Println("[WS]: No listeners, or connection closed")
		return errors.New("[WS]: Connection is nil")
	}
	return nil
}
