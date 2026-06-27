package notify

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type       string          `json:"type"`
	Sequence   int64           `json:"sequence"`
	ServerTime time.Time       `json:"server_time"`
	Payload    json.RawMessage `json:"payload"`
}

type Hub struct {
	mu          sync.RWMutex
	clients     map[*Client]struct{}
	sequence    int64
	subscribers map[string]map[*Client]struct{}
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	subs map[string]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*Client]struct{}),
		subscribers: make(map[string]map[*Client]struct{}),
	}
}

func (h *Hub) Register(conn *websocket.Conn) *Client {
	c := &Client{hub: h, conn: conn, send: make(chan []byte, 64), subs: make(map[string]struct{})}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	go c.writePump()
	return c
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	for ch, subs := range h.subscribers {
		delete(subs, c)
		if len(subs) == 0 {
			delete(h.subscribers, ch)
		}
	}
	h.mu.Unlock()
	close(c.send)
}

func (c *Client) Subscribe(channels []string) {
	c.hub.mu.Lock()
	defer c.hub.mu.Unlock()
	for _, ch := range channels {
		c.subs[ch] = struct{}{}
		if _, ok := c.hub.subscribers[ch]; !ok {
			c.hub.subscribers[ch] = make(map[*Client]struct{})
		}
		c.hub.subscribers[ch][c] = struct{}{}
	}
}

func (h *Hub) Broadcast(channel, msgType string, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	h.mu.Lock()
	h.sequence++
	seq := h.sequence
	subs := h.subscribers[channel]
	h.mu.Unlock()

	msg := Message{Type: msgType, Sequence: seq, ServerTime: time.Now().UTC(), Payload: raw}
	data, _ := json.Marshal(msg)
	for client := range subs {
		select {
		case client.send <- data:
		default:
			go h.dropClient(client)
		}
	}
}

func (h *Hub) dropClient(c *Client) {
	_ = c.conn.Close()
	h.Unregister(c)
}

func (c *Client) writePump() {
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			c.hub.Unregister(c)
			return
		}
	}
}
