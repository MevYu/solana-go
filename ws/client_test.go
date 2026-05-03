package ws

import (
	"context"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWSClient_DialAndClose(t *testing.T) {
	srv, wsURL := startTestWSServer(t, func(c *websocket.Conn) {
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})
	defer srv.Close()

	c, err := DialWebSocket(context.Background(), wsURL)
	if err != nil {
		t.Fatal(err)
	}
	if c.Endpoint() != wsURL {
		t.Errorf("Endpoint = %q, want %q", c.Endpoint(), wsURL)
	}
	_ = c.Close()
	select {
	case <-c.Done():
	case <-time.After(time.Second):
		t.Fatal("Done not closed after Close")
	}
}

func TestWSClient_Dial_InvalidScheme(t *testing.T) {
	if _, err := DialWebSocket(context.Background(), "http://example.com"); err == nil {
		t.Fatal("expected error for http scheme")
	}
}
