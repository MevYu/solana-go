package ws

import (
	"context"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"
	"github.com/MevYu/solana-go/rpc"
)

// ProgramNotification is a single update delivered to a
// ProgramSubscription. Pubkey is the account that changed and
// Account is its new state. Slot is set from the JSON-RPC context
// envelope after decode; the value-object fields decode directly via
// json: tags.
type ProgramNotification struct {
	Slot    uint64              `json:"-"`
	Pubkey  solana.PublicKey    `json:"pubkey"`
	Account *solana.AccountInfo `json:"account"`
}

// ProgramSubscription is the handle returned by ProgramSubscribe.
type ProgramSubscription struct {
	*Subscription
	ch <-chan *ProgramNotification
}

// Recv returns the channel that delivers program-owned account
// updates.
func (s *ProgramSubscription) Recv() <-chan *ProgramNotification { return s.ch }

// ProgramSubscribe subscribes to updates for every account owned
// by the given program. This is a high-volume subscription for
// popular programs; always consume promptly or the buffer will
// drop the oldest notifications.
// Encoding defaults to base64.
func (c *Client) ProgramSubscribe(ctx context.Context, programID solana.PublicKey, cfg ...rpc.CommitmentWithEncodingCfg) (*ProgramSubscription, error) {
	params := []any{programID.String(), buildCommitmentWithEncoding(rpc.FirstOrZero(cfg))}

	ch := make(chan *ProgramNotification, 64)
	codec := c.Codec()
	dispatch := func(raw []byte) {
		var envelope jsonrpc.ContextValue[ProgramNotification]
		if err := codec.Unmarshal(raw, &envelope); err != nil {
			return
		}
		n := envelope.Value
		n.Slot = envelope.Context.Slot
		sendOrDropOldest(ch, &n)
	}
	sub, err := c.Subscribe(ctx, "programSubscribe", "programUnsubscribe", params, dispatch, func() {})
	if err != nil {
		return nil, err
	}
	return &ProgramSubscription{Subscription: sub, ch: ch}, nil
}

// SignatureNotification is the one-shot update delivered to a
// SignatureSubscription once the target signature has reached the
// requested commitment level. Slot is set from the JSON-RPC context
// envelope after decode.
type SignatureNotification struct {
	Slot uint64 `json:"-"`
	Err  any    `json:"err"` // nil on success
}

// SignatureSubscription is the handle returned by SignatureSubscribe.
type SignatureSubscription struct {
	*Subscription
	ch <-chan *SignatureNotification
}

// Recv returns the channel that delivers the single expected
// signature notification. After the notification arrives the
// subscription is done; the server automatically unsubscribes.
func (s *SignatureSubscription) Recv() <-chan *SignatureNotification { return s.ch }

// SignatureSubscribe subscribes to the status of a single
// transaction signature. The server delivers exactly one
// notification once the signature is confirmed (or fails), then
// unsubscribes automatically.
func (c *Client) SignatureSubscribe(ctx context.Context, sig solana.Signature, cfg ...rpc.SignatureSubscribeCfg) (*SignatureSubscription, error) {
	params := []any{sig.String(), rpc.FirstOrZero(cfg)}

	ch := make(chan *SignatureNotification, 4)
	codec := c.Codec()
	dispatch := func(raw []byte) {
		var envelope jsonrpc.ContextValue[SignatureNotification]
		if err := codec.Unmarshal(raw, &envelope); err != nil {
			return
		}
		n := envelope.Value
		n.Slot = envelope.Context.Slot
		sendOrDropOldest(ch, &n)
	}
	sub, err := c.Subscribe(ctx, "signatureSubscribe", "signatureUnsubscribe", params, dispatch, func() {})
	if err != nil {
		return nil, err
	}
	return &SignatureSubscription{Subscription: sub, ch: ch}, nil
}

// BlockNotification is a single update delivered to a
// BlockSubscription. Err is non-nil if the server could not
// reproduce the block for the subscriber; Block is nil in that case.
// Unlike most subscriptions, Slot here decodes from the value-object
// directly (the inner slot) rather than from the context envelope.
type BlockNotification struct {
	Slot  uint64              `json:"slot"`
	Err   any                 `json:"err"`
	Block *rpc.GetBlockResult `json:"block"`
}

// BlockSubscription is the handle returned by BlockSubscribe.
type BlockSubscription struct {
	*Subscription
	ch <-chan *BlockNotification
}

// Recv returns the channel that delivers block notifications.
func (s *BlockSubscription) Recv() <-chan *BlockNotification { return s.ch }

// BlockFilter is the filter passed to BlockSubscribe. Exactly one
// of All or MentionsAccountOrProgram should be set.
type BlockFilter struct {
	// All subscribes to every block.
	All bool
	// MentionsAccountOrProgram restricts the subscription to blocks
	// that contain at least one transaction mentioning this
	// account or program.
	MentionsAccountOrProgram solana.PublicKey
}

// blockSubscribeMentions is the JSON shape for the mentions-filter
// alternative of BlockSubscribe's first argument.
type blockSubscribeMentions struct {
	MentionsAccountOrProgram string `json:"mentionsAccountOrProgram"`
}

// BlockSubscribe subscribes to block updates matching the filter.
// This is an unstable RPC method on mainnet; require-explicit-opt-in
// via --rpc-pubsub-enable-block-subscription on the server.
//
// Encoding defaults to base64, MaxSupportedTransactionVersion to 0,
// TransactionDetails to "full".
func (c *Client) BlockSubscribe(ctx context.Context, filter BlockFilter, cfg ...rpc.GetBlockCfg) (*BlockSubscription, error) {
	var filterValue any
	if !filter.MentionsAccountOrProgram.IsZero() {
		filterValue = blockSubscribeMentions{
			MentionsAccountOrProgram: filter.MentionsAccountOrProgram.String(),
		}
	} else {
		filterValue = "all"
	}

	c0 := *rpc.FirstOrZero(cfg)
	if c0.Encoding == "" {
		c0.Encoding = solana.EncodingBase64
	}
	if c0.TransactionDetails == "" {
		c0.TransactionDetails = "full"
	}
	if c0.MaxSupportedTransactionVersion == nil {
		zero := uint64(0)
		c0.MaxSupportedTransactionVersion = &zero
	}
	params := []any{filterValue, c0}

	ch := make(chan *BlockNotification, 16)
	codec := c.Codec()
	dispatch := func(raw []byte) {
		var envelope jsonrpc.ContextValue[BlockNotification]
		if err := codec.Unmarshal(raw, &envelope); err != nil {
			return
		}
		n := envelope.Value
		sendOrDropOldest(ch, &n)
	}
	sub, err := c.Subscribe(ctx, "blockSubscribe", "blockUnsubscribe", params, dispatch, func() {})
	if err != nil {
		return nil, err
	}
	return &BlockSubscription{Subscription: sub, ch: ch}, nil
}

// SlotUpdate is a single event delivered to a SlotsUpdatesSubscription.
// Type names the kind of event ("firstShredReceived", "completed",
// "createdBank", "frozen", "dead", "optimisticConfirmation", "root").
type SlotUpdate struct {
	Type      string `json:"type"`
	Slot      uint64 `json:"slot"`
	Parent    uint64 `json:"parent,omitempty"`
	Timestamp int64  `json:"timestamp"`
	Err       string `json:"err,omitempty"`
}

// SlotsUpdatesSubscription is the handle returned by
// SlotsUpdatesSubscribe.
type SlotsUpdatesSubscription struct {
	*Subscription
	ch <-chan *SlotUpdate
}

// Recv returns the channel that delivers slot update events.
func (s *SlotsUpdatesSubscription) Recv() <-chan *SlotUpdate { return s.ch }

// SlotsUpdatesSubscribe subscribes to detailed slot lifecycle
// events. Unlike SlotSubscribe, this fires multiple times per slot,
// one event per lifecycle stage (shred received, bank created,
// frozen, rooted, ...).
//
// SlotsUpdatesSubscribe is an unstable RPC method; require explicit
// server-side opt-in via --rpc-pubsub-enable-slots-updates-subscription.
//
// SlotsUpdatesSubscribe does not accept options.
func (c *Client) SlotsUpdatesSubscribe(ctx context.Context) (*SlotsUpdatesSubscription, error) {
	ch := make(chan *SlotUpdate, 64)
	codec := c.Codec()
	dispatch := func(raw []byte) {
		var n SlotUpdate
		if err := codec.Unmarshal(raw, &n); err != nil {
			return
		}
		sendOrDropOldest(ch, &n)
	}
	sub, err := c.Subscribe(ctx, "slotsUpdatesSubscribe", "slotsUpdatesUnsubscribe", nil, dispatch, func() {})
	if err != nil {
		return nil, err
	}
	return &SlotsUpdatesSubscription{Subscription: sub, ch: ch}, nil
}
