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
	var result string
	if err := c.CallContext(ctx, &result, "getHealth"); err != nil {
		return "", err
	}
	return result, nil
}

// GetVersion returns the current version of the Solana node.
func (c *Client) GetVersion(ctx context.Context) (*SolanaVersion, error) {
	var result SolanaVersion
	if err := c.CallContext(ctx, &result, "getVersion"); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetIdentity returns the identity public key of the current node.
func (c *Client) GetIdentity(ctx context.Context) (solana.PublicKey, error) {
	var raw struct {
		Identity string `json:"identity"`
	}
	if err := c.CallContext(ctx, &raw, "getIdentity"); err != nil {
		return solana.PublicKey{}, err
	}
	return solana.PublicKeyFromBase58(raw.Identity)
}

// GetGenesisHash returns the genesis hash.
func (c *Client) GetGenesisHash(ctx context.Context) (solana.Hash, error) {
	var raw string
	if err := c.CallContext(ctx, &raw, "getGenesisHash"); err != nil {
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
	var result HighestSnapshotSlot
	if err := c.CallContext(ctx, &result, "getHighestSnapshotSlot"); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTransactionCount returns the current transaction count from the ledger.
func (c *Client) GetTransactionCount(ctx context.Context, cfg ...CommitmentWithMinSlotCfg) (uint64, error) {
	var count uint64
	if err := c.CallContext(ctx, &count, "getTransactionCount", FirstOrZero(cfg)); err != nil {
		return 0, err
	}
	return count, nil
}

// IsBlockhashValid returns whether a blockhash is still valid.
func (c *Client) IsBlockhashValid(ctx context.Context, blockhash solana.Hash, cfg ...CommitmentWithMinSlotCfg) (bool, error) {
	var resp jsonrpc.ContextValue[bool]
	if err := c.CallContext(ctx, &resp, "isBlockhashValid", blockhash.String(), FirstOrZero(cfg)); err != nil {
		return false, err
	}
	return resp.Value, nil
}

// MinimumLedgerSlot returns the lowest slot that the node has information about.
func (c *Client) MinimumLedgerSlot(ctx context.Context) (uint64, error) {
	var slot uint64
	if err := c.CallContext(ctx, &slot, "minimumLedgerSlot"); err != nil {
		return 0, err
	}
	return slot, nil
}

// GetMaxRetransmitSlot returns the max slot seen from the retransmit stage.
func (c *Client) GetMaxRetransmitSlot(ctx context.Context) (uint64, error) {
	var slot uint64
	if err := c.CallContext(ctx, &slot, "getMaxRetransmitSlot"); err != nil {
		return 0, err
	}
	return slot, nil
}

// GetMaxShredInsertSlot returns the max slot seen from after shred insert.
func (c *Client) GetMaxShredInsertSlot(ctx context.Context) (uint64, error) {
	var slot uint64
	if err := c.CallContext(ctx, &slot, "getMaxShredInsertSlot"); err != nil {
		return 0, err
	}
	return slot, nil
}
