package rpc

import (
	"context"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"
)

// GetBlock fetches the content of a single block by slot.
// Returns nil, nil if the slot is absent or skipped.
//
// SDK-applied defaults when the caller leaves them unset:
//   - Encoding: base64
//   - TransactionDetails: "full"
//   - MaxSupportedTransactionVersion: 0
func (c *Client) GetBlock(ctx context.Context, slot uint64, cfg ...GetBlockCfg) (*GetBlockResult, error) {
	c0 := *FirstOrZero(cfg)
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

	var raw *GetBlockResult
	if err := c.CallContext(ctx, &raw, "getBlock", slot, c0); err != nil {
		return nil, err
	}
	return raw, nil
}

// GetBlocks returns the list of confirmed block slots in the inclusive range [start, end].
// When end is nil, the cluster's latest confirmed block bounds the range.
func (c *Client) GetBlocks(ctx context.Context, start uint64, end *uint64, cfg ...CommitmentCfg) ([]uint64, error) {
	var result []uint64
	if err := c.CallContext(ctx, &result, "getBlocks", start, end, FirstOrZero(cfg)); err != nil {
		return nil, err
	}
	return result, nil
}

// TokenAmount is the common shape used by RPC methods that return SPL Token balances.
type TokenAmount struct {
	Amount         string   `json:"amount"`
	Decimals       uint8    `json:"decimals"`
	UIAmount       *float64 `json:"uiAmount"`
	UIAmountString string   `json:"uiAmountString"`
}

// GetTokenAccountBalanceResult is the decoded response of GetTokenAccountBalance.
type GetTokenAccountBalanceResult struct {
	Slot  uint64
	Value TokenAmount
}

// GetTokenAccountBalance returns the balance of an SPL Token account.
func (c *Client) GetTokenAccountBalance(ctx context.Context, account solana.PublicKey, cfg ...CommitmentCfg) (*GetTokenAccountBalanceResult, error) {
	var resp jsonrpc.ContextValue[TokenAmount]
	if err := c.CallContext(ctx, &resp, "getTokenAccountBalance", account.String(), FirstOrZero(cfg)); err != nil {
		return nil, err
	}
	return &GetTokenAccountBalanceResult{Slot: resp.Context.Slot, Value: resp.Value}, nil
}

// GetTokenSupplyResult is the decoded response of GetTokenSupply.
type GetTokenSupplyResult struct {
	Slot  uint64
	Value TokenAmount
}

// GetTokenSupply returns the total supply of an SPL Token mint.
func (c *Client) GetTokenSupply(ctx context.Context, mint solana.PublicKey, cfg ...CommitmentCfg) (*GetTokenSupplyResult, error) {
	var resp jsonrpc.ContextValue[TokenAmount]
	if err := c.CallContext(ctx, &resp, "getTokenSupply", mint.String(), FirstOrZero(cfg)); err != nil {
		return nil, err
	}
	return &GetTokenSupplyResult{Slot: resp.Context.Slot, Value: resp.Value}, nil
}

// BlockCommitment is the decoded response of GetBlockCommitment.
type BlockCommitment struct {
	Commitment []uint64 `json:"commitment"`
	TotalStake uint64   `json:"totalStake"`
}

// GetBlockCommitment returns the commitment for a given block slot.
func (c *Client) GetBlockCommitment(ctx context.Context, slot uint64) (*BlockCommitment, error) {
	var result BlockCommitment
	if err := c.CallContext(ctx, &result, "getBlockCommitment", slot); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetBlocksWithLimit returns a list of confirmed blocks starting at startSlot up to limit blocks.
func (c *Client) GetBlocksWithLimit(ctx context.Context, startSlot uint64, limit uint64, cfg ...CommitmentCfg) ([]uint64, error) {
	var result []uint64
	if err := c.CallContext(ctx, &result, "getBlocksWithLimit", startSlot, limit, FirstOrZero(cfg)); err != nil {
		return nil, err
	}
	return result, nil
}

// GetFirstAvailableBlock returns the slot of the lowest confirmed block not purged from the ledger.
func (c *Client) GetFirstAvailableBlock(ctx context.Context) (uint64, error) {
	var slot uint64
	if err := c.CallContext(ctx, &slot, "getFirstAvailableBlock"); err != nil {
		return 0, err
	}
	return slot, nil
}
