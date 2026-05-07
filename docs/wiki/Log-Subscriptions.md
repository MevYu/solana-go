# Log Subscriptions

`LogsSubscribe` streams the program log messages that each
transaction emits. It is the observability lever for Solana
programs: every `msg!` macro in an on-chain program lands on
this stream.

## API

```go
func (c *ws.Client) LogsSubscribe(
    ctx context.Context,
    filter ws.LogsFilter,
    cfg ...rpc.LogsSubscribeCfg,
) (*ws.LogsSubscription, error)

type LogsFilter struct {
    All          bool
    AllWithVotes bool
    Mentions     []solana.PublicKey
}

func (s *LogsSubscription) Recv() <-chan *LogNotification

type LogNotification struct {
    Slot      uint64
    Signature string // base58
    Err       any    // nil on success
    Logs      []string
}
```

Exactly one of `All`, `AllWithVotes`, or `Mentions` should be
set; if none are set, the filter defaults to `All`.

Cfg: `rpc.LogsSubscribeCfg{Commitment: …}`.

## Filter semantics

- **`All`** — every non-vote transaction. Low-ish volume
  compared to the full stream.
- **`AllWithVotes`** — literally every transaction, votes
  included. High volume; expect thousands per second on
  mainnet.
- **`Mentions`** — transactions that mention at least one of
  the given addresses. Solana currently allows only a single
  pubkey per subscription, so treat `Mentions` as a one-element
  slice.

## Example: watch a wallet's logs

```go
import (
    "github.com/MevYu/solana-go"
    "github.com/MevYu/solana-go/rpc"
)

wsc, _ := ws.DialWebSocket(ctx, endpoint)
defer wsc.Close()

wallet, _ := solana.PublicKeyFromBase58("...")
sub, err := wsc.LogsSubscribe(ctx, ws.LogsFilter{
    Mentions: []solana.PublicKey{wallet},
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
        if n.Err != nil {
            fmt.Printf("%s: FAIL %v\n", n.Signature, n.Err)
        } else {
            fmt.Printf("%s: ok\n", n.Signature)
        }
        for _, line := range n.Logs {
            fmt.Printf("  %s\n", line)
        }
    }
}
```

## Example: watch an Anchor program for errors

```go
programID, _ := solana.PublicKeyFromBase58("...")
sub, _ := wsc.LogsSubscribe(ctx, ws.LogsFilter{
    Mentions: []solana.PublicKey{programID},
})

for {
    select {
    case <-sub.Done():
        return
    case n := <-sub.Recv():
        if n.Err == nil {
            continue
        }
        err := rpc.DecodeTransactionError(n.Err)
        log.Printf("[%s] %v", n.Signature, err)
    }
}
```

`rpc.DecodeTransactionError` is the same decoder used in
simulation — it produces typed `*InstructionError` values you
can match with `errors.As`.

## Back-pressure

64-slot buffer with drop-oldest. For high-volume filters
(`All`, `AllWithVotes`), be prepared to miss intermediate
notifications if your consumer falls behind the read loop.

## Vote transactions

Vote transactions are the bulk of Solana's throughput and are
almost never interesting for application-level observability.
`All` excludes them by default; use `AllWithVotes` only if
you explicitly need to watch the consensus layer.

## Related

- [Simulate With Decoded Errors](Simulate-With-Decoded-Errors)
  — how to turn `n.Err` into a typed Go error.
- [Signature Subscriptions](Advanced-Subscriptions#signaturesubscribe)
  — for a one-shot "did my transaction land" notification.
