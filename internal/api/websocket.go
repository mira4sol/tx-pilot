package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func (deps Dependencies) websocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := deps.Hub.Register(conn)
	defer deps.Hub.Unregister(client)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var sub struct {
			Type     string   `json:"type"`
			Channels []string `json:"channels"`
		}
		if err := json.Unmarshal(msg, &sub); err == nil && sub.Type == "subscribe" {
			client.Subscribe(sub.Channels)
		}
	}
}
