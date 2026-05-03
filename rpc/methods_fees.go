package rpc

import (
	"context"
	"encoding/base64"
	"fmt"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"
)

// RequestAirdrop asks the cluster to deposit lamports into pubkey.
// Only valid on devnet and testnet.
func (c *Client) RequestAirdrop(ctx context.Context, pubkey solana.PublicKey, lamports uint64, cfg ...CommitmentCfg) (solana.Signature, error) {
	var sigStr string
	if err := c.CallContext(ctx, &sigStr, "requestAirdrop", pubkey.String(), lamports, FirstOrZero(cfg)); err != nil {
		return solana.Signature{}, err
	}
	return solana.SignatureFromBase58(sigStr)
}

// GetFeeForMessageResult is the decoded response of GetFeeForMessage.
type GetFeeForMessageResult struct {
	Slot uint64
	Fee  *uint64
}

// GetFeeForMessage computes the fee the cluster would charge for a transaction with the given message.
func (c *Client) GetFeeForMessage(ctx context.Context, msg *solana.Message, cfg ...CommitmentWithMinSlotCfg) (*GetFeeForMessageResult, error) {
	if msg == nil {
		return nil, fmt.Errorf("solana: GetFeeForMessage: nil message")
	}
	msgBytes, err := msg.Marshal()
	if err != nil {
		return nil, fmt.Errorf("solana: GetFeeForMessage: marshal: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(msgBytes)
	slot, fee, err := jsonrpc.CallContextValue[*uint64](ctx, c.Client, "getFeeForMessage", encoded, FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return &GetFeeForMessageResult{Slot: slot, Fee: fee}, nil
}

// MaxGetRecentPrioritizationFeesAddresses is the per-request limit
// the Solana RPC server enforces on getRecentPrioritizationFees.
const MaxGetRecentPrioritizationFeesAddresses = 128

// GetRecentPrioritizationFees returns the prioritization fees observed in recent slots.
func (c *Client) GetRecentPrioritizationFees(ctx context.Context, addresses []solana.PublicKey) ([]PrioritizationFee, error) {
	if len(addresses) > MaxGetRecentPrioritizationFeesAddresses {
		return nil, fmt.Errorf("solana: GetRecentPrioritizationFees: %d addresses exceeds Solana RPC max of %d", len(addresses), MaxGetRecentPrioritizationFeesAddresses)
	}
	keys := make([]string, len(addresses))
	for i, a := range addresses {
		keys[i] = a.String()
	}
	var result []PrioritizationFee
	if err := c.CallContext(ctx, &result, "getRecentPrioritizationFees", keys); err != nil {
		return nil, err
	}
	return result, nil
}
