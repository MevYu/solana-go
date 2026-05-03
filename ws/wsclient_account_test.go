package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	solana "github.com/MevYu/solana-go"

	"github.com/gorilla/websocket"
)

func TestWsClient_AccountSubscribe(t *testing.T) {
	srv, wsURL := startTestWSServer(t, func(c *websocket.Conn) {
		_, msg, err := c.ReadMessage()
		if err != nil {
			return
		}
		var req struct {
			ID     uint64 `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(msg, &req); err != nil {
			t.Errorf("unmarshal request: %v", err)
			return
		}
		if req.Method != "accountSubscribe" {
			t.Errorf("method = %q", req.Method)
		}

		resp := fmt.Sprintf(`{"jsonrpc":"2.0","result":12345,"id":%d}`, req.ID)
		if err := c.WriteMessage(websocket.TextMessage, []byte(resp)); err != nil {
			return
		}

		notif := `{"jsonrpc":"2.0","method":"accountNotification","params":{"subscription":12345,"result":{"context":{"slot":100},"value":{"lamports":2039280,"owner":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","data":["",""],"executable":false,"rentEpoch":0,"space":0}}}}`
		if err := c.WriteMessage(websocket.TextMessage, []byte(notif)); err != nil {
			return
		}

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

	pk, _ := solana.PublicKeyFromBase58(tokenProgramID)
	sub, err := c.AccountSubscribe(ctx, pk)
	if err != nil {
		t.Fatal(err)
	}
	if sub.ID() != 12345 {
		t.Errorf("ID = %d, want 12345", sub.ID())
	}

	select {
	case n := <-sub.Recv():
		if n.Slot != 100 {
			t.Errorf("slot = %d", n.Slot)
		}
		if n.Value == nil || n.Value.Lamports != 2039280 {
			t.Errorf("value mismatch: %+v", n.Value)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for notification")
	}

	if err := sub.Unsubscribe(ctx); err != nil {
		t.Errorf("Unsubscribe: %v", err)
	}
	select {
	case <-sub.Done():
	case <-time.After(time.Second):
		t.Fatal("Done not closed after Unsubscribe")
	}
}
