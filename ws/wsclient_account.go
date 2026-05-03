package ws

import (
	"github.com/MevYu/solana-go/jsonrpc"
	"context"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/rpc"
)

// AccountNotification is a single update delivered to an
// AccountSubscription.
type AccountNotification struct {
	// Slot is the slot at which the update was observed.
	Slot uint64
	// Value is the new account state, or nil if the account was
	// closed or is otherwise absent at the reported slot.
	Value *solana.AccountInfo
}

// AccountSubscription is the handle returned by AccountSubscribe.
// Read notifications from Recv(); stop the subscription with
// Unsubscribe. Because Recv() is not closed on shutdown, callers
// should also select on Done() to detect termination.
type AccountSubscription struct {
	*Subscription
	ch <-chan *AccountNotification
}

// Recv returns the channel that delivers account notifications.
// The channel is intentionally never closed; detect the end of the
// subscription via Done() instead.
func (s *AccountSubscription) Recv() <-chan *AccountNotification { return s.ch }

// commitmentWithEncodingParams is the JSON-RPC param object shared by
// accountSubscribe and programSubscribe. Encoding is always sent
// (base64 by default); commitment is omitted when unset.
type commitmentWithEncodingParams struct {
	Commitment rpc.CommitmentLevel `json:"commitment,omitempty"`
	Encoding   rpc.Encoding        `json:"encoding"`
}

// buildCommitmentWithEncoding fills the shared params struct and
// defaults the encoding to base64 when unset.
func buildCommitmentWithEncoding(cfg *rpc.CommitmentWithEncodingCfg) commitmentWithEncodingParams {
	p := commitmentWithEncodingParams{
		Commitment: cfg.Commitment,
		Encoding:   cfg.Encoding,
	}
	if p.Encoding == "" {
		p.Encoding = rpc.EncodingBase64
	}
	return p
}

// AccountSubscribe subscribes to updates for the given account. It
// returns once the server has acknowledged the subscribe request.
// Encoding defaults to base64.
func (c *Client) AccountSubscribe(ctx context.Context, pubkey solana.PublicKey, cfg ...rpc.CommitmentWithEncodingCfg) (*AccountSubscription, error) {
	params := []any{pubkey.String(), buildCommitmentWithEncoding(rpc.FirstOrZero(cfg))}

	ch := make(chan *AccountNotification, 64)
	codec := c.Codec()

	dispatch := func(raw []byte) {
		var envelope jsonrpc.ContextValue[*solana.AccountInfo]
		if err := codec.Unmarshal(raw, &envelope); err != nil {
			return
		}
		n := &AccountNotification{
			Slot:  envelope.Context.Slot,
			Value: envelope.Value,
		}
		sendOrDropOldest(ch, n)
	}
	shutdown := func() {
		// ch is intentionally not closed; see type doc.
	}

	sub, err := c.Subscribe(ctx, "accountSubscribe", "accountUnsubscribe", params, dispatch, shutdown)
	if err != nil {
		return nil, err
	}
	return &AccountSubscription{Subscription: sub, ch: ch}, nil
}
