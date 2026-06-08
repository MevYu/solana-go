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

func TestWsClient_ProgramSubscribe(t *testing.T) {
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
		if req.Method != "programSubscribe" {
			t.Errorf("method = %q", req.Method)
		}
		resp := fmt.Sprintf(`{"jsonrpc":"2.0","result":55,"id":%d}`, req.ID)
		_ = c.WriteMessage(websocket.TextMessage, []byte(resp))

		notif := `{"jsonrpc":"2.0","method":"programNotification","params":{"subscription":55,"result":{"context":{"slot":200},"value":{"pubkey":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","account":{"lamports":1000,"owner":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","data":["","base64"],"executable":false,"rentEpoch":0,"space":0}}}}}`
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

	pk, _ := solana.PublicKeyFromBase58(tokenProgramID)
	sub, err := c.ProgramSubscribe(ctx, pk)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case n := <-sub.Recv():
		if n.Slot != 200 {
			t.Errorf("slot = %d", n.Slot)
		}
		if n.Account == nil || n.Account.Lamports != 1000 {
			t.Errorf("account mismatch: %+v", n.Account)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	_ = sub.Unsubscribe(ctx)
}

func TestWsClient_SignatureSubscribe(t *testing.T) {
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
		if req.Method != "signatureSubscribe" {
			t.Errorf("method = %q", req.Method)
		}
		resp := fmt.Sprintf(`{"jsonrpc":"2.0","result":11,"id":%d}`, req.ID)
		_ = c.WriteMessage(websocket.TextMessage, []byte(resp))

		notif := `{"jsonrpc":"2.0","method":"signatureNotification","params":{"subscription":11,"result":{"context":{"slot":321},"value":{"err":null}}}}`
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

	tx, kps := makeSimpleTx(t, 1)
	_ = tx.Sign(ctx, kps[0])
	sub, err := c.SignatureSubscribe(ctx, tx.Signatures[0])
	if err != nil {
		t.Fatal(err)
	}
	select {
	case n := <-sub.Recv():
		if n.Slot != 321 {
			t.Errorf("slot = %d", n.Slot)
		}
		if n.Err != nil {
			t.Errorf("Err should be nil, got %+v", n.Err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	// One-shot subscriptions auto-unsubscribe server-side after the single
	// notification; the local entry must be dropped too (no map leak).
	<-sub.Done()
	c.mu.RLock()
	_, leaked := c.subscriptions[sub.ID()]
	n := len(c.subscriptions)
	c.mu.RUnlock()
	if leaked || n != 0 {
		t.Fatalf("signature subscription leaked local entry: present=%v, map size=%d", leaked, n)
	}

	_ = sub.Unsubscribe(ctx)
}

func TestWsClient_SlotsUpdatesSubscribe(t *testing.T) {
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
		if req.Method != "slotsUpdatesSubscribe" {
			t.Errorf("method = %q", req.Method)
		}
		resp := fmt.Sprintf(`{"jsonrpc":"2.0","result":9,"id":%d}`, req.ID)
		_ = c.WriteMessage(websocket.TextMessage, []byte(resp))

		notif := `{"jsonrpc":"2.0","method":"slotsUpdatesNotification","params":{"subscription":9,"result":{"type":"frozen","slot":100,"parent":99,"timestamp":1700000000}}}`
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

	sub, err := c.SlotsUpdatesSubscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case u := <-sub.Recv():
		if u.Type != "frozen" || u.Slot != 100 || u.Parent != 99 {
			t.Errorf("slot update mismatch: %+v", u)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	_ = sub.Unsubscribe(ctx)
}

func TestWsClient_BlockSubscribe(t *testing.T) {
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
		if req.Method != "blockSubscribe" {
			t.Errorf("method = %q", req.Method)
		}
		resp := fmt.Sprintf(`{"jsonrpc":"2.0","result":77,"id":%d}`, req.ID)
		_ = c.WriteMessage(websocket.TextMessage, []byte(resp))

		notif := `{"jsonrpc":"2.0","method":"blockNotification","params":{"subscription":77,"result":{"context":{"slot":500},"value":{"slot":500,"err":null,"block":{"blockhash":"HUhwFvKfmVhZ5AbkqK7N2Kt3i6hPz3jQeByN6wA9RvXb","previousBlockhash":"8MG4BQ5C2zR7ADvptZ4hG4rGvXfjzeLPmKtPPvzqS3M9","parentSlot":499,"blockHeight":480,"blockTime":1700000000,"rewards":[],"transactions":[]}}}}}`
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

	sub, err := c.BlockSubscribe(ctx, BlockFilter{All: true})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case n := <-sub.Recv():
		if n.Slot != 500 {
			t.Errorf("slot = %d", n.Slot)
		}
		if n.Block == nil || n.Block.ParentSlot != 499 {
			t.Errorf("block mismatch: %+v", n.Block)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	_ = sub.Unsubscribe(ctx)
}
