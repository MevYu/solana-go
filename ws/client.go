// Package ws provides the low-level WebSocket transport for Solana
// JSON-RPC pub/sub notifications. It manages the connection, the
// read loop, and the subscribe/unsubscribe plumbing. Typed, per-API
// subscriptions (AccountSubscribe, LogsSubscribe, …) live on
// *client.Client in the client package.
package ws

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MevYu/solana-go/jsonrpc"

	"github.com/gorilla/websocket"
)

// Client manages a WebSocket connection to a Solana RPC endpoint
// and fans out subscription notifications to per-subscription typed
// channels.
//
// A Client maintains a single underlying WebSocket connection
// shared by all subscriptions. Construct one with DialWebSocket;
// call Close to tear down the connection and stop every active
// subscription.
//
// Client is safe for concurrent use by multiple goroutines.
type Client struct {
	endpoint string
	conn     *websocket.Conn
	codec    jsonrpc.Codec

	writeMu sync.Mutex

	mu            sync.RWMutex
	nextReqID     uint64
	pendingReqs   map[uint64]*pendingReply
	subscriptions map[uint64]*subEntry
	closedFlag    bool

	done chan struct{}
	once sync.Once
	err  atomic.Value // stores error
}

// pendingReply carries a subscribe request's state from the
// caller goroutine to the read loop and back. It is constructed
// by Subscribe, stored in pendingReqs keyed by request id, and
// resolved by dispatchIncoming when the server replies.
//
// Crucially, dispatchIncoming registers the resulting subscription
// in subscriptions map BEFORE closing done. This closes the race
// where the server sends a subscribe reply and a notification
// back-to-back: by the time the read loop pulls the notification
// off the wire, the subscription is already in the map and the
// notification routes correctly.
type pendingReply struct {
	// Set by Subscribe:
	sub          *Subscription
	dispatch     func([]byte)
	userShutdown func()
	// oneShot subscriptions (e.g. signatureSubscribe) get exactly one
	// notification and are then auto-unsubscribed server-side; the read
	// loop drops their local entry after delivering it.
	oneShot bool

	// Set by dispatchIncoming:
	errMsg string

	// Signal: closed by dispatchIncoming once registration is done
	// (or an error is recorded in errMsg).
	done chan struct{}
}

// subEntry is the internal bookkeeping for an active subscription.
type subEntry struct {
	subID    uint64
	dispatch func([]byte)
	shutdown func()
	oneShot  bool
}

// DialWebSocket connects to the given Solana WebSocket RPC
// endpoint. The endpoint must use the ws:// or wss:// scheme.
func DialWebSocket(ctx context.Context, endpoint string) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("solana: ws: parse endpoint: %w", err)
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return nil, fmt.Errorf("solana: ws: endpoint scheme must be ws or wss, got %q", u.Scheme)
	}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("solana: ws: dial: %w", err)
	}
	c := &Client{
		endpoint:      endpoint,
		conn:          conn,
		codec:         jsonrpc.GoJSONCodec(),
		pendingReqs:   make(map[uint64]*pendingReply),
		subscriptions: make(map[uint64]*subEntry),
		done:          make(chan struct{}),
	}
	go c.readLoop()
	return c, nil
}

// Endpoint returns the endpoint URL the client was connected to.
func (c *Client) Endpoint() string { return c.endpoint }

// Codec returns the JSON codec used to encode requests and decode
// notifications. Typed subscriptions in the client package use it
// to Unmarshal notification payloads.
func (c *Client) Codec() jsonrpc.Codec { return c.codec }

// Done returns a channel that is closed when the client has
// terminated, either by Close or a fatal connection error.
func (c *Client) Done() <-chan struct{} { return c.done }

// Err returns the error that terminated the client, or nil if the
// client is still running or was closed gracefully.
func (c *Client) Err() error {
	v := c.err.Load()
	if v == nil {
		return nil
	}
	return v.(error)
}

// Close terminates the WebSocket connection and all active
// subscriptions. It is safe to call Close multiple times.
func (c *Client) Close() error {
	var closeErr error
	c.once.Do(func() {
		_ = c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(2*time.Second),
		)
		closeErr = c.conn.Close()

		c.mu.Lock()
		c.closedFlag = true
		pending := c.pendingReqs
		c.pendingReqs = nil
		subs := c.subscriptions
		c.subscriptions = nil
		c.mu.Unlock()

		// Unblock any in-flight subscribe calls with an error.
		for _, p := range pending {
			p.errMsg = "connection closed"
			close(p.done)
		}
		// Tear down active subscriptions.
		for _, e := range subs {
			e.shutdown()
		}
		close(c.done)
	})
	return closeErr
}

// readLoop reads incoming WebSocket frames and dispatches them to
// either a pending request channel or an active subscription.
func (c *Client) readLoop() {
	defer c.Close()
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			c.err.Store(err)
			return
		}
		c.dispatchIncoming(msg)
	}
}

// wsEnvelope is the union shape of every incoming frame: a reply (ID set)
// or a notification (Method set, Params populated). Params is a value, not
// a pointer, so decoding a notification doesn't heap-allocate it per frame;
// replies leave it zero. Routing keys on ID presence, not Params.
type wsEnvelope struct {
	ID     *uint64         `json:"id"`
	Method string          `json:"method"`
	Result jsonrpc.RawJSON `json:"result"`
	Error  *jsonrpc.Error  `json:"error"`
	Params struct {
		Subscription uint64          `json:"subscription"`
		Result       jsonrpc.RawJSON `json:"result"`
	} `json:"params"`
}

func (c *Client) dispatchIncoming(msg []byte) {
	var envelope wsEnvelope
	if err := c.codec.Unmarshal(msg, &envelope); err != nil {
		return
	}
	if envelope.ID != nil {
		c.handleReply(*envelope.ID, envelope.Result, envelope.Error)
		return
	}
	if envelope.Method == "" {
		return
	}
	subID := envelope.Params.Subscription
	c.mu.RLock()
	entry, ok := c.subscriptions[subID]
	c.mu.RUnlock()
	if !ok {
		return
	}
	entry.dispatch([]byte(envelope.Params.Result))
	if entry.oneShot {
		// Server auto-unsubscribes after this single notification; drop our
		// local entry too so the subscriptions map doesn't grow unbounded
		// under high signature-confirm churn.
		c.mu.Lock()
		if c.subscriptions != nil {
			delete(c.subscriptions, subID)
		}
		c.mu.Unlock()
		entry.shutdown()
	}
}

// handleReply resolves a subscribe or unsubscribe response. For a
// successful subscribe, it REGISTERS the subscription in
// subscriptions map before closing pending.done. This guarantees
// that any notification arriving on the same read loop cycle
// immediately after the reply will find its entry.
func (c *Client) handleReply(id uint64, result jsonrpc.RawJSON, rpcErr *jsonrpc.Error) {
	c.mu.Lock()
	pending, ok := c.pendingReqs[id]
	if ok {
		delete(c.pendingReqs, id)
	}
	c.mu.Unlock()
	if !ok {
		return
	}

	if rpcErr != nil {
		pending.errMsg = rpcErr.Message
		close(pending.done)
		return
	}

	var subID uint64
	if err := c.codec.Unmarshal(result, &subID); err != nil {
		// Unsubscribe acks return a bool, not a uint64. We don't
		// track unsubscribe reply handlers (Unsubscribe is
		// fire-and-forget), so this branch is only reachable if
		// something routes a bool-returning method through
		// Subscribe; record it as an error.
		pending.errMsg = "decode subscription id: " + err.Error()
		close(pending.done)
		return
	}

	sub := pending.sub
	sub.id = subID

	userShutdown := pending.userShutdown
	shutdownFn := func() {
		sub.once.Do(func() { close(sub.subDone) })
		userShutdown()
	}

	c.mu.Lock()
	if c.subscriptions == nil {
		// Closed between our delete-from-pendingReqs and here;
		// abort the subscription.
		c.mu.Unlock()
		pending.errMsg = "client closed"
		close(pending.done)
		return
	}
	c.subscriptions[subID] = &subEntry{
		subID:    subID,
		dispatch: pending.dispatch,
		shutdown: shutdownFn,
		oneShot:  pending.oneShot,
	}
	c.mu.Unlock()

	close(pending.done)
}

// Subscribe sends a subscribe request and blocks until the server
// acknowledges. By the time Subscribe returns, the subscription is
// already registered in the subscriptions map (by handleReply), so
// any notifications that arrive immediately after the reply are
// routed correctly.
//
// method and unsubMethod are the JSON-RPC method names
// (e.g. "accountSubscribe" and "accountUnsubscribe"). params is
// the request's params array. dispatch is invoked for every
// incoming notification payload. shutdown is invoked once when
// the subscription ends (via Unsubscribe or client Close).
func (c *Client) Subscribe(
	ctx context.Context,
	method, unsubMethod string,
	params []any,
	dispatch func([]byte),
	shutdown func(),
) (*Subscription, error) {
	return c.subscribe(ctx, method, unsubMethod, params, dispatch, shutdown, false)
}

// subscribe is Subscribe with an explicit one-shot flag. A one-shot
// subscription receives exactly one notification and is auto-unsubscribed
// server-side; the read loop drops its local entry after delivery.
func (c *Client) subscribe(
	ctx context.Context,
	method, unsubMethod string,
	params []any,
	dispatch func([]byte),
	shutdown func(),
	oneShot bool,
) (*Subscription, error) {
	c.mu.Lock()
	if c.closedFlag {
		c.mu.Unlock()
		return nil, errors.New("solana: ws: client is closed")
	}
	c.nextReqID++
	reqID := c.nextReqID

	sub := &Subscription{
		client:      c,
		method:      method,
		unsubMethod: unsubMethod,
		subDone:     make(chan struct{}),
	}
	pending := &pendingReply{
		sub:          sub,
		dispatch:     dispatch,
		userShutdown: shutdown,
		oneShot:      oneShot,
		done:         make(chan struct{}),
	}
	c.pendingReqs[reqID] = pending
	c.mu.Unlock()

	req := jsonrpc.Request{
		Version: "2.0",
		ID:      reqID,
		Method:  method,
		Params:  params,
	}
	body, err := c.codec.Marshal(&req)
	if err != nil {
		c.cancelPending(reqID)
		return nil, fmt.Errorf("solana: ws: marshal request: %w", err)
	}

	c.writeMu.Lock()
	err = c.conn.WriteMessage(websocket.TextMessage, body)
	c.writeMu.Unlock()
	if err != nil {
		c.cancelPending(reqID)
		return nil, fmt.Errorf("solana: ws: write: %w", err)
	}

	select {
	case <-ctx.Done():
		c.cancelPending(reqID)
		return nil, ctx.Err()
	case <-c.done:
		return nil, errors.New("solana: ws: client closed before reply")
	case <-pending.done:
	}
	if pending.errMsg != "" {
		return nil, fmt.Errorf("solana: ws: %s: %s", method, pending.errMsg)
	}
	return sub, nil
}

func (c *Client) cancelPending(reqID uint64) {
	c.mu.Lock()
	delete(c.pendingReqs, reqID)
	c.mu.Unlock()
}

// Subscription is the handle returned by Subscribe. It exposes the
// server-assigned id, a Done channel that fires when the
// subscription ends, and an Unsubscribe method. Typed subscriptions
// in the client package embed *Subscription to inherit these.
type Subscription struct {
	client      *Client
	id          uint64
	method      string
	unsubMethod string
	subDone     chan struct{}
	once        sync.Once
}

// ID returns the server-assigned subscription id.
func (s *Subscription) ID() uint64 { return s.id }

// Done returns a channel that is closed when the subscription ends,
// either because Unsubscribe was called or because the underlying
// client was closed.
func (s *Subscription) Done() <-chan struct{} { return s.subDone }

// Unsubscribe cancels the subscription and releases server-side
// resources. It is safe to call Unsubscribe multiple times;
// subsequent calls are no-ops.
func (s *Subscription) Unsubscribe(ctx context.Context) error {
	s.client.mu.Lock()
	entry, ok := s.client.subscriptions[s.id]
	if ok {
		delete(s.client.subscriptions, s.id)
	}
	closed := s.client.closedFlag
	s.client.nextReqID++
	reqID := s.client.nextReqID
	s.client.mu.Unlock()

	if entry != nil {
		entry.shutdown()
	}
	if closed {
		return nil
	}

	// Fire-and-forget: don't wait for the unsubscribe ack. If the
	// server is slow, we still release our side of the channel
	// bookkeeping immediately, and the server times out the dangling
	// subscription on its own schedule.
	req := jsonrpc.Request{
		Version: "2.0",
		ID:      reqID,
		Method:  s.unsubMethod,
		Params:  []any{s.id},
	}
	body, err := s.client.codec.Marshal(&req)
	if err != nil {
		return fmt.Errorf("solana: ws: unsubscribe marshal: %w", err)
	}
	s.client.writeMu.Lock()
	err = s.client.conn.WriteMessage(websocket.TextMessage, body)
	s.client.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("solana: ws: unsubscribe write: %w", err)
	}
	return nil
}
