# Slot Subscriptions

Two slot-oriented subscriptions live in `ws/wsclient_slots.go`:
`SlotSubscribe` for slot progress and `RootSubscribe` for root
progress.

## `SlotSubscribe`

```go
func (c *ws.Client) SlotSubscribe(ctx context.Context) (*SlotSubscription, error)

type SlotInfo struct {
    Parent uint64
    Root   uint64
    Slot   uint64
}
```

Fires once per newly processed slot. `Slot` is the slot just
processed, `Parent` is its parent slot, and `Root` is the
cluster's current root slot.

Use `SlotSubscribe` for:

- Driving progress bars / latency dashboards.
- Deriving approximate wall-clock time from slot deltas (~400ms
  per slot on mainnet).
- Implementing a blockhash refresh loop without polling
  `GetBlockHeight`.

Accepts no options.

### Example

```go
sub, err := wsc.SlotSubscribe(ctx)
if err != nil {
    return err
}
defer sub.Unsubscribe(context.Background())

for {
    select {
    case <-sub.Done():
        return nil
    case info := <-sub.Recv():
        fmt.Printf("slot=%d parent=%d root=%d\n",
            info.Slot, info.Parent, info.Root)
    }
}
```

## `RootSubscribe`

```go
func (c *ws.Client) RootSubscribe(ctx context.Context) (*RootSubscription, error)
func (s *RootSubscription) Recv() <-chan uint64
```

Fires each time the cluster advances its root (the oldest slot
every supermajority fork contains). The notification is just
the new root slot as a `uint64`.

Use `RootSubscribe` when you want the strongest guarantee that
data will never roll back: once a slot is rooted, every
supermajority fork has it, so any transaction landing in that
slot is permanent.

Accepts no options.

### Example

```go
sub, err := wsc.RootSubscribe(ctx)
if err != nil {
    return err
}
defer sub.Unsubscribe(context.Background())

for {
    select {
    case <-sub.Done():
        return nil
    case root := <-sub.Recv():
        fmt.Printf("new root: %d\n", root)
    }
}
```

## Back-pressure

Both subscriptions use a 64-slot buffer with drop-oldest. Slot
updates fire at ~2.5 Hz on mainnet, so the buffer is rarely
under pressure unless the consumer is severely backed up.

## Ordering

Slot updates arrive in slot order. Fork switches can cause a
slot to be re-delivered with a different `Parent` — the second
delivery reflects the cluster's newly-chosen fork.

## Related

- [Advanced Subscriptions](Advanced-Subscriptions) —
  `SlotsUpdatesSubscribe` for detailed per-slot lifecycle
  events (first shred, completed, frozen, rooted, ...).
- [Chain Info Methods](Chain-Info-Methods) — `GetSlot` for a
  point-in-time read.
