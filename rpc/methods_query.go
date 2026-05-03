package rpc

import (
	"context"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"
)

// GetBalanceResult is the decoded response of GetBalance.
type GetBalanceResult struct {
	Slot  uint64
	Value uint64
}

// GetBalance returns the balance of the account at the given address, in lamports.
func (c *Client) GetBalance(ctx context.Context, pubkey solana.PublicKey, cfg ...CommitmentWithMinSlotCfg) (*GetBalanceResult, error) {
	slot, lamports, err := jsonrpc.CallContextValue[uint64](ctx, c.Client, "getBalance", pubkey.String(), FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return &GetBalanceResult{Slot: slot, Value: lamports}, nil
}

// GetSlot returns the current slot the node is processing.
func (c *Client) GetSlot(ctx context.Context, cfg ...CommitmentWithMinSlotCfg) (uint64, error) {
	var slot uint64
	if err := c.CallContext(ctx, &slot, "getSlot", FirstOrZero(cfg)); err != nil {
		return 0, err
	}
	return slot, nil
}

// GetBlockHeight returns the current block height.
func (c *Client) GetBlockHeight(ctx context.Context, cfg ...CommitmentWithMinSlotCfg) (uint64, error) {
	var height uint64
	if err := c.CallContext(ctx, &height, "getBlockHeight", FirstOrZero(cfg)); err != nil {
		return 0, err
	}
	return height, nil
}

// GetLatestBlockhash returns the latest blockhash and its last valid block height.
func (c *Client) GetLatestBlockhash(ctx context.Context, cfg ...CommitmentWithMinSlotCfg) (*LatestBlockhash, error) {
	slot, v, err := jsonrpc.CallContextValue[LatestBlockhash](ctx, c.Client, "getLatestBlockhash", FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	v.Slot = slot
	return &v, nil
}
