//go:build integration

package ws_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mira4sol/tx-pilot/test/helpers"
)

func TestWebSocketSubscribe(t *testing.T) {
	base := helpers.BaseURL()
	wsURL := strings.Replace(base, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	u, err := url.Parse(wsURL + "/v1/ws")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer conn.Close()
	if resp.StatusCode != 101 {
		t.Fatalf("expected 101 switching protocols, got %d", resp.StatusCode)
	}

	sub := map[string]any{
		"type": "subscribe",
		"channels": []string{
			"network.ticker",
			"slots.feed",
			"transactions.stream",
			"ai.decisions",
		},
	}
	if err := conn.WriteJSON(sub); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}

	// Trigger a transaction to increase chance of stream events.
	_ = helpers.SubmitDeveloperTx(t, "integration-ws")

	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	for i := 0; i < 3; i++ {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Logf("read message %d: %v (may be idle until broadcast)", i, err)
			break
		}
		var envelope map[string]any
		if err := json.Unmarshal(msg, &envelope); err != nil {
			t.Fatalf("unmarshal ws message: %v", err)
		}
		if _, ok := envelope["type"]; !ok {
			t.Fatalf("ws message missing type: %s", string(msg))
		}
		t.Logf("ws message type=%v", envelope["type"])
	}
}
