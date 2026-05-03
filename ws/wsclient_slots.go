package ws

import (
	"context"
)

// SlotInfo is a single update delivered to a SlotSubscription.
type SlotInfo struct {
	Parent uint64 `json:"parent"`
	Root   uint64 `json:"root"`
	Slot   uint64 `json:"slot"`
}

// SlotSubscription is the handle returned by SlotSubscribe.
type SlotSubscription struct {
	*Subscription
	ch <-chan *SlotInfo
}

// Recv returns the channel that delivers slot updates.
func (s *SlotSubscription) Recv() <-chan *SlotInfo { return s.ch }

// SlotSubscribe subscribes to slot updates. Each notification
// reports the current parent, root, and the newly processed slot.
//
// SlotSubscribe does not accept options.
func (c *Client) SlotSubscribe(ctx context.Context) (*SlotSubscription, error) {
	ch := make(chan *SlotInfo, 64)
	codec := c.Codec()
	dispatch := func(raw []byte) {
		var n SlotInfo
		if err := codec.Unmarshal(raw, &n); err != nil {
			return
		}
		sendOrDropOldest(ch, &n)
	}
	sub, err := c.Subscribe(ctx, "slotSubscribe", "slotUnsubscribe", nil, dispatch, func() {})
	if err != nil {
		return nil, err
	}
	return &SlotSubscription{Subscription: sub, ch: ch}, nil
}

// RootSubscription is the handle returned by RootSubscribe.
type RootSubscription struct {
	*Subscription
	ch <-chan uint64
}

// Recv returns the channel that delivers root slot updates.
func (s *RootSubscription) Recv() <-chan uint64 { return s.ch }

// RootSubscribe subscribes to root updates. Each notification is
// the slot of a newly rooted block (the oldest slot every
// supermajority fork contains).
//
// RootSubscribe does not accept options.
func (c *Client) RootSubscribe(ctx context.Context) (*RootSubscription, error) {
	ch := make(chan uint64, 64)
	codec := c.Codec()
	dispatch := func(raw []byte) {
		var slot uint64
		if err := codec.Unmarshal(raw, &slot); err != nil {
			return
		}
		sendOrDropOldest(ch, slot)
	}
	sub, err := c.Subscribe(ctx, "rootSubscribe", "rootUnsubscribe", nil, dispatch, func() {})
	if err != nil {
		return nil, err
	}
	return &RootSubscription{Subscription: sub, ch: ch}, nil
}
