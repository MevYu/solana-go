package ws

import (
	"github.com/MevYu/solana-go/jsonrpc"
	"context"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/rpc"
)

// LogNotification is a single update delivered to a LogsSubscription.
// Slot comes from the JSON-RPC context envelope; the rest decode
// directly from the value object.
type LogNotification struct {
	// Slot is the slot at which the transaction was processed.
	Slot uint64 `json:"-"`
	// Signature is the base58-encoded transaction signature.
	Signature string `json:"signature"`
	// Err is the transaction error, or nil on success.
	Err any `json:"err"`
	// Logs is the program log messages emitted by the transaction.
	Logs []string `json:"logs"`
}

// LogsFilter is the filter passed to LogsSubscribe. Exactly one of
// All, AllWithVotes, or Mentions should be set; if none are set,
// the filter defaults to All.
type LogsFilter struct {
	// All subscribes to every transaction except vote transactions.
	All bool
	// AllWithVotes subscribes to every transaction including votes.
	AllWithVotes bool
	// Mentions restricts the subscription to transactions that
	// mention at least one of these addresses. Solana currently
	// limits this to a single pubkey per subscription.
	Mentions []solana.PublicKey
}

// LogsSubscription is the handle returned by LogsSubscribe.
type LogsSubscription struct {
	*Subscription
	ch <-chan *LogNotification
}

// Recv returns the channel that delivers log notifications.
func (s *LogsSubscription) Recv() <-chan *LogNotification { return s.ch }

// logsMentionsFilter is the JSON shape for the mentions-filter
// alternative of LogsSubscribe's first argument.
type logsMentionsFilter struct {
	Mentions []string `json:"mentions"`
}

// LogsSubscribe subscribes to transaction logs matching the given
// filter.
func (c *Client) LogsSubscribe(ctx context.Context, filter LogsFilter, cfg ...rpc.LogsSubscribeCfg) (*LogsSubscription, error) {
	var filterValue any
	switch {
	case filter.AllWithVotes:
		filterValue = "allWithVotes"
	case len(filter.Mentions) > 0:
		mentions := make([]string, len(filter.Mentions))
		for i, pk := range filter.Mentions {
			mentions[i] = pk.String()
		}
		filterValue = logsMentionsFilter{Mentions: mentions}
	default:
		filterValue = "all"
	}

	params := []any{filterValue, rpc.FirstOrZero(cfg)}

	ch := make(chan *LogNotification, 64)
	codec := c.Codec()
	dispatch := func(raw []byte) {
		var envelope jsonrpc.ContextValue[LogNotification]
		if err := codec.Unmarshal(raw, &envelope); err != nil {
			return
		}
		n := envelope.Value
		n.Slot = envelope.Context.Slot
		sendOrDropOldest(ch, &n)
	}
	sub, err := c.Subscribe(ctx, "logsSubscribe", "logsUnsubscribe", params, dispatch, func() {})
	if err != nil {
		return nil, err
	}
	return &LogsSubscription{Subscription: sub, ch: ch}, nil
}
