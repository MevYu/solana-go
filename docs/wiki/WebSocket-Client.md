# WebSocket Client

`ws.Client` is the WebSocket companion to `rpc.Client`. It
opens one WebSocket connection to a Solana RPC endpoint and
multiplexes many subscriptions over it, delivering typed
notifications on per-subscription Go channels.

## Package

The entire WebSocket stack lives in a single top-level package:

| Package | Purpose |
|---|---|
| `ws` | WebSocket transport + typed Subscribe API: `Client`, `DialWebSocket`, `AccountSubscribe`, `LogsSubscribe`, `SlotSubscribe`, `RootSubscribe`, `ProgramSubscribe`, `SignatureSubscribe`, `BlockSubscribe`, `SlotsUpdatesSubscribe`. |

Cfg structs (e.g. `rpc.LogsSubscribeCfg`, `rpc.SignatureSubscribeCfg`,
`rpc.GetBlockCfg`) are imported from `rpc` since they are shared with
the HTTP-side methods.

## Construction

```go
import "github.com/MevYu/solana-go/ws"

wsc, err := ws.DialWebSocket(ctx, "wss://api.mainnet-beta.solana.com")
if err != nil {
    return err
}
defer wsc.Close()
```

The endpoint must use `ws://` or `wss://`. `DialWebSocket` returns
once the handshake completes; it does not block on the first
subscription.

The HTTP `*rpc.Client` and the `*ws.Client` are independent values:
construct each separately and keep both alongside whatever context
object your application uses.

```go
c := rpc.NewClient("https://api.mainnet-beta.solana.com", jsonrpc.Config{})
wsc, _ := ws.DialWebSocket(ctx, "wss://api.mainnet-beta.solana.com")
defer wsc.Close()
// pass `c` and `wsc` together where you need them; there is no
// `c.WithWebSocket(wsc)` / `c.WS()` attachment.
```

## Lifecycle

A `WsClient` runs a single background read loop that dispatches
incoming frames to either a pending request channel or an active
subscription. Its exit channels let you detect termination:

```go
select {
case <-ws.Done():
    if err := ws.Err(); err != nil {
        log.Printf("ws terminated: %v", err)
    }
}
```

`ws.Close()` is idempotent and safe to call from any goroutine.

## Subscribing is synchronous

Every `Subscribe` method blocks until the server acknowledges the
subscribe request. By the time `Subscribe` returns, the subscription
is already registered in the client's dispatch table, so
notifications that arrive immediately after the ack are routed
correctly — no lost updates between "server started streaming" and
"Go-side handler installed".

This ordering guarantee is enforced by the `ws` transport: registration
into the subscription map happens **before** unblocking the caller
goroutine.

## The `Subscription` handle

Every `Subscribe` method returns a typed subscription value whose
embedded `*ws.Subscription` exposes:

```go
func (s *Subscription) ID() uint64                    // server-assigned subscription id
func (s *Subscription) Done() <-chan struct{}         // closed on end
func (s *Subscription) Unsubscribe(ctx context.Context) error
```

- **`ID`** — the subscription id the server picked. Useful for
  logging.
- **`Done`** — closed when the subscription ends (either explicitly
  via `Unsubscribe` or because the WS connection terminated).
- **`Unsubscribe`** — fires the unsubscribe method and releases the
  client's internal bookkeeping. Idempotent. Fire-and-forget on the
  wire: it does not wait for the server's unsubscribe ack, so a
  slow server never wedges the caller.

## Back-pressure: drop-oldest

Every typed subscription exposes a buffered channel via its
`Recv()` method. The buffer sizes are tuned per method (64 for
account / log / program / slot, 16 for block, 4 for signature).

**When the buffer is full, the dispatcher drops the oldest buffered
notification and enqueues the new one.** This keeps the read loop
flowing when a slow consumer falls behind. The consumer is never
blocked waiting for the producer, and the producer is never blocked
waiting for the consumer — but consumers that cannot keep up will
lose history.

If you cannot tolerate dropped updates, rewrite your handler to
drain the channel immediately into an unbounded internal queue you
control.

## Channels are never closed

The subscription channels are intentionally **never closed** on
shutdown. Ranging over a subscription channel with
`for x := range sub.Recv()` would block forever after the
subscription ends, so the idiom is a `select` on both the channel
and `Done()`:

```go
for {
    select {
    case <-sub.Done():
        return
    case n := <-sub.Recv():
        handle(n)
    }
}
```

## Notification decoding

Every `*Notification` struct carries `json:` tags directly on its
public fields and is decoded straight from the wire by
`jsonrpc.ContextValue[T]` — the same envelope type that drives the
HTTP-side `jsonrpc.CallContextValue[T]` helper. Slot fields delivered
by the JSON-RPC context envelope are tagged `json:"-"` and assigned
post-decode; the inner-value Slot in `BlockNotification` is tagged
`json:"slot"`.

## Available subscriptions

| Method | Notification | Page |
|---|---|---|
| `AccountSubscribe` | Account state update | [Account Subscriptions](Account-Subscriptions) |
| `LogsSubscribe` | Program log lines per tx | [Log Subscriptions](Log-Subscriptions) |
| `SlotSubscribe` | Slot progress | [Slot Subscriptions](Slot-Subscriptions) |
| `RootSubscribe` | New root slot | [Slot Subscriptions](Slot-Subscriptions) |
| `ProgramSubscribe` | Every account owned by a program | [Advanced Subscriptions](Advanced-Subscriptions) |
| `SignatureSubscribe` | One-shot notification for a signature | [Advanced Subscriptions](Advanced-Subscriptions) |
| `BlockSubscribe` | Block-level updates (opt-in on server) | [Advanced Subscriptions](Advanced-Subscriptions) |
| `SlotsUpdatesSubscribe` | Detailed slot lifecycle events (opt-in) | [Advanced Subscriptions](Advanced-Subscriptions) |

## Error classes

Dial errors (`endpoint scheme must be ws or wss`, `dial failed`)
are returned from `DialWebSocket`.

Subscribe errors (server rejected the subscribe request) are
returned from the individual `Subscribe` methods and are prefixed
with the method name: `solana: ws: accountSubscribe: invalid address`.

Reads that fail inside the background loop are stored on the
client; call `ws.Err()` after `<-ws.Done()` fires to see the root
cause.

## Related

- [RPC Client](RPC-Client) — the HTTP companion.
- [Send Transaction](Send-Transaction) — often used with
  [Signature Subscriptions](Advanced-Subscriptions#signaturesubscribe)
  for low-latency confirmation.
