# Advanced Subscriptions

Four specialised subscriptions live in `ws/wsclient_extra.go`:

- `ProgramSubscribe`
- `SignatureSubscribe`
- `BlockSubscribe`
- `SlotsUpdatesSubscribe`

## `ProgramSubscribe`

```go
func (c *ws.Client) ProgramSubscribe(
    ctx context.Context,
    programID PublicKey,
    opts ...CallOption,
) (*ProgramSubscription, error)

type ProgramNotification struct {
    Slot    uint64
    Pubkey  PublicKey    // the account that changed
    Account *AccountInfo // its new state
}
```

Streams updates for **every** account owned by the given
program. This is a high-volume subscription for popular
programs (SPL Token, Jupiter, Raydium); prefer
`AccountSubscribe` on a specific account if you can.

Honoured options: `WithCommitment`, `WithEncoding`
(default base64).

### Example: watch every USDC token account change

```go
tokenProgram, _ := solana.PublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
sub, err := ws.ProgramSubscribe(ctx, tokenProgram,
    rpc.WithCommitment(solana.CommitmentConfirmed),
)
```

This fires on **every** SPL Token account update on mainnet,
which is thousands per second — expect to drop updates unless
you route to a downstream queue.

## `SignatureSubscribe`

```go
func (c *ws.Client) SignatureSubscribe(
    ctx context.Context,
    sig Signature,
    opts ...CallOption,
) (*SignatureSubscription, error)

type SignatureNotification struct {
    Slot uint64
    Err  any // nil on success
}
```

Fires **exactly once**, when the signature is confirmed (or
fails) at the requested commitment. After the notification,
the server automatically unsubscribes.

This is the lowest-latency way to wait for confirmation of a
specific transaction — the server pushes as soon as the
signature reaches commitment, no polling required.

Honoured options: `WithCommitment`.

### Example: confirm a just-submitted transaction

```go
sig, _ := c.SendTransaction(ctx, tx)

sub, err := ws.SignatureSubscribe(ctx, sig,
    rpc.WithCommitment(solana.CommitmentConfirmed),
)
if err != nil {
    return err
}

select {
case <-sub.Done():
    return fmt.Errorf("subscription ended before notification")
case n := <-sub.Recv():
    if n.Err != nil {
        return rpc.DecodeTransactionError(n.Err)
    }
    fmt.Printf("confirmed in slot %d\n", n.Slot)
}
```

## `BlockSubscribe`

```go
func (c *ws.Client) BlockSubscribe(
    ctx context.Context,
    filter BlockFilter,
    opts ...CallOption,
) (*BlockSubscription, error)

type BlockFilter struct {
    All                      bool
    MentionsAccountOrProgram PublicKey
}

type BlockNotification struct {
    Slot  uint64
    Err   any
    Block *GetBlockResult
}
```

**Unstable RPC method on mainnet.** The server must be
launched with `--rpc-pubsub-enable-block-subscription` to
accept the subscribe request.

Honoured options: `WithCommitment`, `WithEncoding`,
`WithMaxSupportedTransactionVersion`.

When a subscription is set up with `MentionsAccountOrProgram`,
the server filters for blocks containing at least one
transaction that mentions that account or program. With
`All`, every block is delivered.

Expect high bandwidth per notification — each `Block` is a
fully-materialised `GetBlockResult`. Set a data-slice-like
commitment (`Processed` is fastest) and budget memory
accordingly.

## `SlotsUpdatesSubscribe`

```go
func (c *ws.Client) SlotsUpdatesSubscribe(ctx context.Context) (*SlotsUpdatesSubscription, error)

type SlotUpdate struct {
    Type      string // "firstShredReceived", "completed", "createdBank", "frozen", "dead", "optimisticConfirmation", "root"
    Slot      uint64
    Parent    uint64
    Timestamp int64
    Err       string
}
```

**Unstable RPC method.** The server must be launched with
`--rpc-pubsub-enable-slots-updates-subscription` to accept
the subscribe request.

Unlike `SlotSubscribe`, this method fires multiple times per
slot — one event per lifecycle stage. It is useful for
cluster instrumentation and slot-by-slot latency analysis.

Accepts no options.

### Example: measure shred-receive to frozen latency

```go
sub, _ := ws.SlotsUpdatesSubscribe(ctx)
defer sub.Unsubscribe(context.Background())

firstSeen := map[uint64]int64{}

for {
    select {
    case <-sub.Done():
        return
    case u := <-sub.Recv():
        switch u.Type {
        case "firstShredReceived":
            firstSeen[u.Slot] = u.Timestamp
        case "frozen":
            if t0, ok := firstSeen[u.Slot]; ok {
                fmt.Printf("slot %d: %dms to frozen\n", u.Slot, u.Timestamp-t0)
                delete(firstSeen, u.Slot)
            }
        }
    }
}
```

## Related

- [WebSocket Client](WebSocket-Client) — the hosting client.
- [Signature Methods](Signature-Methods) — the HTTP
  equivalent of `SignatureSubscribe`.
- [Simulate With Decoded Errors](Simulate-With-Decoded-Errors)
  — decoding the `Err` field shared across many notifications.
