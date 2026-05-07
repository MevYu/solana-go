# Account Subscriptions

`AccountSubscribe` streams updates for a single account address.
It is the most common WebSocket subscription and the right way
to maintain a live view of a wallet or program state without
polling.

## API

```go
func (c *ws.Client) AccountSubscribe(
    ctx context.Context,
    pubkey solana.PublicKey,
    cfg ...rpc.CommitmentWithEncodingCfg,
) (*ws.AccountSubscription, error)

type AccountSubscription struct {
    *ws.Subscription
    // ...
}

func (s *AccountSubscription) Recv() <-chan *AccountNotification

type AccountNotification struct {
    Slot  uint64
    Value *solana.AccountInfo // nil if the account was closed
}
```

## Cfg fields (`rpc.CommitmentWithEncodingCfg`)

- `Commitment`
- `Encoding` (default: `EncodingBase64`)

## Example

```go
import (
    "github.com/MevYu/solana-go"
    "github.com/MevYu/solana-go/rpc"
    "github.com/MevYu/solana-go/ws"
)

wsc, err := ws.DialWebSocket(ctx, "wss://api.mainnet-beta.solana.com")
if err != nil {
    return err
}
defer wsc.Close()

wallet, _ := solana.PublicKeyFromBase58("...")
sub, err := wsc.AccountSubscribe(ctx, wallet, rpc.CommitmentWithEncodingCfg{
    Commitment: solana.CommitmentConfirmed,
})
if err != nil {
    return err
}
defer sub.Unsubscribe(context.Background())

for {
    select {
    case <-sub.Done():
        return nil
    case n := <-sub.Recv():
        if n.Value == nil {
            fmt.Printf("slot %d: account closed\n", n.Slot)
            continue
        }
        fmt.Printf("slot %d: %d lamports, %d data bytes\n",
            n.Slot, n.Value.Lamports, len(n.Value.Data.Value()))
    }
}
```

## Decoding the account data

`n.Value.Data` is the standard [AccountData](Accounts)
two-element shape. Call `Bytes()` to decode, then pass to
your program's binary decoder:

```go
raw, err := n.Value.Data.Bytes()
if err != nil {
    log.Printf("unsupported encoding %q", n.Value.Data.Encoding())
    continue
}
// raw is the post-update account data
```

## Back-pressure

`AccountSubscription` uses a 64-slot buffered channel. When the
buffer fills, the dispatcher **drops the oldest buffered
notification** and enqueues the new one. This keeps the read
loop flowing at the cost of losing stale intermediate updates.

For watching a hot account (large AMM pool, high-traffic
vault), budget either:

1. A fast in-process handler that re-exports to a larger queue
   you control; or
2. A coarser commitment (`Processed` produces fewer updates
   than `Confirmed`); or
3. A data slice (if the account is huge) to reduce per-update
   bytes.

## Ordering and correctness

- Notifications arrive in slot order, but the cluster can
  occasionally replay a slot on fork switching.
- `AccountSubscribe` fires on any state change, including
  lamport-only changes (transfer in / out) and ownership
  changes — not just data changes.
- A `Value == nil` notification means the account was closed
  or does not exist at the reported slot.

## Unsubscribe

```go
_ = sub.Unsubscribe(context.Background())
```

Fire-and-forget; the background loop and the server release
their bookkeeping. Idempotent.

## Related

- [Advanced Subscriptions](Advanced-Subscriptions) —
  `ProgramSubscribe` for all accounts owned by a program in
  one shot.
- [Account Methods](Account-Methods) — point-in-time fetch.
- [WebSocket Client](WebSocket-Client) — the client that
  hosts this subscription.
