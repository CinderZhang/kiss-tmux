package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

//go:embed web
var webFS embed.FS

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.Mutex
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// safeSend sends data to the client's send channel without panicking
// if the channel has been closed by the hub.
func (c *Client) safeSend(data []byte) {
	defer func() { recover() }()
	select {
	case c.send <- data:
	default:
	}
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump(hub *Hub, mgr *Manager) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512 * 1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg ClientMsg
		if err := c.conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("ws error: %v", err)
			}
			break
		}

		switch msg.Type {
		case "spawn":
			id, err := mgr.Spawn(msg.Cmd, msg.Cwd, msg.Name, msg.Cols, msg.Rows)
			if err != nil {
				errMsg := ServerMsg{Type: "error", Error: err.Error()}
				if data, err := json.Marshal(errMsg); err == nil {
					c.safeSend(data)
				}
			} else {
				spawnedMsg := ServerMsg{Type: "spawned", Session: id}
				if data, err := json.Marshal(spawnedMsg); err == nil {
					c.safeSend(data)
				}
			}
		case "input":
			if err := mgr.Input(msg.Session, msg.Data); err != nil {
				errMsg := ServerMsg{Type: "error", Session: msg.Session, Error: err.Error()}
				if data, err := json.Marshal(errMsg); err == nil {
					c.safeSend(data)
				}
			}
		case "resize":
			mgr.Resize(msg.Session, msg.Cols, msg.Rows)
		case "kill":
			mgr.Kill(msg.Session)
		case "rename":
			mgr.Rename(msg.Session, msg.Name)
		case "list":
			sessMsg := mgr.SessionsMessage()
			if data, err := json.Marshal(sessMsg); err == nil {
				c.safeSend(data)
			}
		}
	}
}

func sendInitialState(client *Client, mgr *Manager) {
	sessMsg := mgr.SessionsMessage()
	if data, err := json.Marshal(sessMsg); err == nil {
		client.safeSend(data)
	}

	for _, s := range mgr.List() {
		replay := mgr.Replay(s.ID)
		if len(replay) > 0 {
			msg := ServerMsg{Type: "output", Session: s.ID, Data: string(replay)}
			if data, err := json.Marshal(msg); err == nil {
				client.safeSend(data)
			}
		}
	}
}

func SetupHTTP(hub *Hub, mgr *Manager) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade error: %v", err)
			return
		}

		client := &Client{
			hub:  hub,
			conn: conn,
			send: make(chan []byte, 256),
		}
		go client.writePump()

		hub.register <- client
		sendInitialState(client, mgr)

		client.readPump(hub, mgr)
	})

	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	return mux
}
