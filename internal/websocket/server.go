package websocket

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

/*
Server - websocket server
*/
type Server struct {
	url         string
	route       string
	hb          time.Duration
	isDebug     bool
	upgrader    websocket.Upgrader
	dataChannel DataChannel
	subsChannel SubsChannel

	sync.Mutex

	connections   map[string]*Connection
	subscriptions map[string]map[string]*Connection
}

type DataChannel chan Message
type SubsChannel chan Subscription
type Subscription struct {
	isAdd   bool
	channel string
	conn    *Connection
}

// NewServer - websocket server constructor
func NewServer(url string, hb int, route string) *Server {
	return &Server{
		url:           url,
		route:         route,
		hb:            time.Duration(time.Duration(hb) * time.Second),
		isDebug:       isDebug(),
		connections:   make(map[string]*Connection),
		subscriptions: make(map[string]map[string]*Connection),
		subsChannel:   make(SubsChannel),
	}
}

func (ws *Server) SetChannel(ch DataChannel) {
	ws.dataChannel = ch
}

// Start - starting ws server
func (ws *Server) Start() error {
	log.Println("Starting websocket server on ", ws.url)
	ws.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	ws.handleSubscriptions()

	http.HandleFunc(ws.route, ws.handler)
	err := http.ListenAndServe(ws.url, nil)
	if err != nil {
		return NewServerStartError(err)
	}
	return nil // Hm...
}

func (ws *Server) Stop() {
	log.Println("ws.connections: ", ws.connections)
	for _, c := range ws.connections {
		log.Printf("c: %+v\n", c)
		if c.conn != nil {
			c.conn.Close()
		}
	}
}
func (ws *Server) Broadcast(ch string, data interface{}) {
	subs := ws.subscriptions[ch]
	if subs == nil || len(subs) == 0 {
		return
	}
	if ws.isDebug {
		// fmt.Println("[WS][Broadcast] Channel: ", ch, len(subs))
	}

	for _, conn := range subs {
		conn.Write(data)
	}
}

func (ws *Server) handleSubscriptions() {
	go func() {
		for sub := range ws.subsChannel {
			if sub.isAdd {
				ws.subscribe(sub.channel, sub.conn)
			} else {
				ws.unsubscribe(sub.channel, sub.conn)
			}
		}
	}()
}

func (ws *Server) subscribe(channel string, conn *Connection) error {

	ws.Lock()
	defer ws.Unlock()
	if ws.isDebug {
		fmt.Println("=========== Subscribe: before ===========")
		fmt.Println("Channel: ", ws.subscriptions[channel])
		fmt.Println("Len: ", len(ws.subscriptions))
	}
	ch := ws.subscriptions[channel]

	if ch == nil {
		ch = make(map[string]*Connection, 0)
	}

	ch[conn.ID] = conn
	ws.subscriptions[channel] = ch
	if ws.isDebug {
		fmt.Println("=========== Subscribe: after ===========")
		fmt.Println("Channel: ", ws.subscriptions[channel])
		fmt.Println("Len: ", len(ws.subscriptions))
	}

	return nil
}
func (ws *Server) unsubscribe(channel string, conn *Connection) error {
	ws.Lock()
	defer ws.Unlock()
	if ws.isDebug {
		fmt.Println("=========== unsubscribe: before ===========")
		fmt.Println("Channel: ", ws.subscriptions[channel])
		fmt.Println("Len: ", len(ws.subscriptions[channel]))
	}
	ch := ws.subscriptions[channel]

	if ch == nil {
		return nil
	}
	delete(ws.subscriptions[channel], conn.ID)
	if ws.isDebug {
		fmt.Println("=========== unsubscribe: after ===========")
		fmt.Println("Channel: ", ws.subscriptions[channel])
		fmt.Println("Len: ", len(ws.subscriptions[channel]))
	}
	return nil
}

func (ws *Server) handler(w http.ResponseWriter, r *http.Request) {
	if ws.isDebug {
		fmt.Println("New request", r.URL)
	}

	c, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		ws.handleError(err)
		return
	}
	conn := NewConnection(c, r, ws.dataChannel, ws.subsChannel)
	ws.addConnection(conn)
	defer func() {
		fmt.Println("Disconnect: ", conn.ID)
		ws.closeConnection(conn.ID)
	}()

	conn.Listen()
}

func (ws *Server) addConnection(conn *Connection) {
	ws.connections[conn.ID] = conn
}

func (ws *Server) closeConnection(connID string) error {
	if ws.isDebug {
		fmt.Println("Disconnecting: ", connID, len(ws.connections))
	}
	ws.unsubscribeConnection(connID)
	ws.Lock()
	defer ws.Unlock()
	if _, ok := ws.connections[connID]; ok {
		delete(ws.connections, connID)
	}
	if ws.isDebug {
		fmt.Println("Disconnected: ", connID, len(ws.connections))
	}
	return nil

}

func (ws *Server) unsubscribeConnection(connID string) error {
	if c, ok := ws.connections[connID]; ok {
		for channel := range c.subs {
			ws.unsubscribe(channel, c)
		}
	}
	return nil
}

func (ws *Server) handleError(err error) {
	if err != nil {
		log.Println("Error:", err)
	}
}

func isDebug() (debug bool) {
	if os.Getenv("DEBUG") == "true" {
		debug = true
	}
	return
}
