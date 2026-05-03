package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWsClient_SlotSubscribe(t *testing.T) {
	srv, wsURL := startTestWSServer(t, func(c *websocket.Conn) {
		_, msg, err := c.ReadMessage()
		if err != nil {
			return
		}
		var req struct {
			ID     uint64 `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(msg, &req)
		if req.Method != "slotSubscribe" {
			t.Errorf("method = %q", req.Method)
		}
		resp := fmt.Sprintf(`{"jsonrpc":"2.0","result":7,"id":%d}`, req.ID)
		_ = c.WriteMessage(websocket.TextMessage, []byte(resp))

		notif := `{"jsonrpc":"2.0","method":"slotNotification","params":{"subscription":7,"result":{"parent":99,"root":98,"slot":100}}}`
		_ = c.WriteMessage(websocket.TextMessage, []byte(notif))

		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})
	defer srv.Close()

	ctx := context.Background()
	c, err := DialWebSocket(ctx, wsURL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	sub, err := c.SlotSubscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case n := <-sub.Recv():
		if n.Slot != 100 || n.Parent != 99 || n.Root != 98 {
			t.Errorf("slot info mismatch: %+v", n)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	_ = sub.Unsubscribe(ctx)
}

func TestWsClient_RootSubscribe(t *testing.T) {
	srv, wsURL := startTestWSServer(t, func(c *websocket.Conn) {
		_, msg, err := c.ReadMessage()
		if err != nil {
			return
		}
		var req struct {
			ID     uint64 `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(msg, &req)
		if req.Method != "rootSubscribe" {
			t.Errorf("method = %q", req.Method)
		}
		resp := fmt.Sprintf(`{"jsonrpc":"2.0","result":3,"id":%d}`, req.ID)
		_ = c.WriteMessage(websocket.TextMessage, []byte(resp))

		notif := `{"jsonrpc":"2.0","method":"rootNotification","params":{"subscription":3,"result":12345}}`
		_ = c.WriteMessage(websocket.TextMessage, []byte(notif))

		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})
	defer srv.Close()

	ctx := context.Background()
	c, err := DialWebSocket(ctx, wsURL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	sub, err := c.RootSubscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case slot := <-sub.Recv():
		if slot != 12345 {
			t.Errorf("slot = %d", slot)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	_ = sub.Unsubscribe(ctx)
}

func TestWsClient_LogsSubscribe(t *testing.T) {
	srv, wsURL := startTestWSServer(t, func(c *websocket.Conn) {
		_, msg, err := c.ReadMessage()
		if err != nil {
			return
		}
		var req struct {
			ID     uint64 `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(msg, &req)
		if req.Method != "logsSubscribe" {
			t.Errorf("method = %q", req.Method)
		}
		resp := fmt.Sprintf(`{"jsonrpc":"2.0","result":42,"id":%d}`, req.ID)
		_ = c.WriteMessage(websocket.TextMessage, []byte(resp))

		notif := `{"jsonrpc":"2.0","method":"logsNotification","params":{"subscription":42,"result":{"context":{"slot":500},"value":{"signature":"5Nqm","err":null,"logs":["Program 11111 invoke","Program 11111 success"]}}}}`
		_ = c.WriteMessage(websocket.TextMessage, []byte(notif))

		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})
	defer srv.Close()

	ctx := context.Background()
	c, err := DialWebSocket(ctx, wsURL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	sub, err := c.LogsSubscribe(ctx, LogsFilter{All: true})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case n := <-sub.Recv():
		if n.Slot != 500 {
			t.Errorf("slot = %d", n.Slot)
		}
		if n.Signature != "5Nqm" {
			t.Errorf("signature = %q", n.Signature)
		}
		if len(n.Logs) != 2 {
			t.Errorf("logs len = %d", len(n.Logs))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	_ = sub.Unsubscribe(ctx)
}
