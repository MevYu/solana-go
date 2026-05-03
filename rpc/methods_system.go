package rpc

import (
	"context"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"
)

// SolanaVersion is the decoded response of GetVersion.
type SolanaVersion struct {
	SolanaCore string `json:"solana-core"`
	FeatureSet uint32 `json:"feature-set"`
}

// GetHealth returns the current health of the node.
func (c *Client) GetHealth(ctx context.Context) (string, error) {
	result, err := jsonrpc.CallContext[string](ctx, c.Client, "getHealth")
	if err != nil {
		return "", err
	}
	return result, nil
}

// GetVersion returns the current version of the Solana node.
func (c *Client) GetVersion(ctx context.Context) (*SolanaVersion, error) {
	result, err := jsonrpc.CallContext[SolanaVersion](ctx, c.Client, "getVersion")
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetIdentity returns the identity public key of the current node.
func (c *Client) GetIdentity(ctx context.Context) (solana.PublicKey, error) {
	raw, err := jsonrpc.CallContext[struct {
		Identity string `json:"identity"`
	}](ctx, c.Client, "getIdentity")
	if err != nil {
		return solana.PublicKey{}, err
	}
	return solana.PublicKeyFromBase58(raw.Identity)
}

// GetGenesisHash returns the genesis hash.
func (c *Client) GetGenesisHash(ctx context.Context) (solana.Hash, error) {
	raw, err := jsonrpc.CallContext[string](ctx, c.Client, "getGenesisHash")
	if err != nil {
		return solana.Hash{}, err
	}
	return solana.HashFromBase58(raw)
}

// HighestSnapshotSlot is the decoded response of GetHighestSnapshotSlot.
type HighestSnapshotSlot struct {
	Full        uint64  `json:"full"`
	Incremental *uint64 `json:"incremental"`
}

// GetHighestSnapshotSlot returns the highest slot for which the node has a snapshot.
func (c *Client) GetHighestSnapshotSlot(ctx context.Context) (*HighestSnapshotSlot, error) {
	result, err := jsonrpc.CallContext[HighestSnapshotSlot](ctx, c.Client, "getHighestSnapshotSlot")
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTransactionCount returns the current transaction count from the ledger.
func (c *Client) GetTransactionCount(ctx context.Context, cfg ...CommitmentWithMinSlotCfg) (uint64, error) {
	count, err := jsonrpc.CallContext[uint64](ctx, c.Client, "getTransactionCount", FirstOrZero(cfg))
	if err != nil {
		return 0, err
	}
	return count, nil
}

// IsBlockhashValid returns whether a blockhash is still valid.
func (c *Client) IsBlockhashValid(ctx context.Context, blockhash solana.Hash, cfg ...CommitmentWithMinSlotCfg) (bool, error) {
	resp, err := jsonrpc.CallContext[jsonrpc.ContextValue[bool]](ctx, c.Client, "isBlockhashValid", blockhash.String(), FirstOrZero(cfg))
	if err != nil {
		return false, err
	}
	return resp.Value, nil
}

// MinimumLedgerSlot returns the lowest slot that the node has information about.
func (c *Client) MinimumLedgerSlot(ctx context.Context) (uint64, error) {
	slot, err := jsonrpc.CallContext[uint64](ctx, c.Client, "minimumLedgerSlot")
	if err != nil {
		return 0, err
	}
	return slot, nil
}

// GetMaxRetransmitSlot returns the max slot seen from the retransmit stage.
func (c *Client) GetMaxRetransmitSlot(ctx context.Context) (uint64, error) {
	slot, err := jsonrpc.CallContext[uint64](ctx, c.Client, "getMaxRetransmitSlot")
	if err != nil {
		return 0, err
	}
	return slot, nil
}

// GetMaxShredInsertSlot returns the max slot seen from after shred insert.
func (c *Client) GetMaxShredInsertSlot(ctx context.Context) (uint64, error) {
	slot, err := jsonrpc.CallContext[uint64](ctx, c.Client, "getMaxShredInsertSlot")
	if err != nil {
		return 0, err
	}
	return slot, nil
}
