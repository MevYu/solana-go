// Package rpc provides the typed Solana JSON-RPC client. It embeds a
// jsonrpc.Client transport and exposes Solana-specific methods
// (GetAccountInfo, SendTransaction, …). For raw JSON-RPC transport
// only — without the Solana method surface — use the jsonrpc package
// directly.
//
// WebSocket subscriptions live in the ws package; attach a *ws.Client
// to a *Client via WithWebSocket when you need both RPC and WS on one
// logical client.
package rpc

import (
	"github.com/MevYu/solana-go/jsonrpc"
)

// Client is the typed Solana JSON-RPC client. It embeds *jsonrpc.Client
// so the underlying transport's Call / batching / retry surface is
// promoted directly.
type Client struct {
	*jsonrpc.Client
}

// NewClient returns a Client connected to the given JSON-RPC endpoint
// using a Config struct (HTTP transport options).
//
//	c := rpc.NewClient("https://api.mainnet-beta.solana.com", jsonrpc.Config{
//	    MaxIdleConnsPerHost: 20,
//	})
func NewClient(endpoint string, cfg jsonrpc.Config) *Client {
	return &Client{Client: jsonrpc.NewClient(endpoint, cfg)}
}

// NewClientWith returns a Client configured via functional options.
//
//	c := rpc.NewClientWith("https://api.mainnet-beta.solana.com",
//	    jsonrpc.WithMaxIdleConnsPerHost(20),
//	    jsonrpc.WithHeader("X-API-Key", "secret"),
//	)
func NewClientWith(endpoint string, opts ...jsonrpc.ClientOption) *Client {
	return &Client{Client: jsonrpc.NewClientWith(endpoint, opts...)}
}

// NewClientFromTransport wraps an existing *jsonrpc.Client. Use this
// when you need to share a transport (e.g. one decorated with retry,
// rate-limit, or metrics middleware) across multiple Client instances.
func NewClientFromTransport(j *jsonrpc.Client) *Client {
	return &Client{Client: j}
}

// Transport returns the underlying *jsonrpc.Client.
func (c *Client) Transport() *jsonrpc.Client { return c.Client }
