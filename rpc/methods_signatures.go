package rpc

import (
	"context"
	"fmt"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"
)

// MaxGetSignatureStatusesSignatures is the per-request limit the
// Solana RPC server enforces on getSignatureStatuses. The SDK
// validates input length up-front so callers get a precise error
// instead of a vague "Invalid params" from the server.
const MaxGetSignatureStatusesSignatures = 256

// GetSignatureStatuses fetches the status of multiple transaction signatures.
// Set cfg.SearchTransactionHistory to query the long-term transaction archive.
func (c *Client) GetSignatureStatuses(ctx context.Context, sigs []solana.Signature, cfg ...SignatureStatusesCfg) (*GetSignatureStatusesResult, error) {
	if len(sigs) > MaxGetSignatureStatusesSignatures {
		return nil, fmt.Errorf("solana: GetSignatureStatuses: %d signatures exceeds Solana RPC max of %d", len(sigs), MaxGetSignatureStatusesSignatures)
	}
	sigStrs := make([]string, len(sigs))
	for i, s := range sigs {
		sigStrs[i] = s.String()
	}
	resp, err := jsonrpc.CallContext[jsonrpc.ContextValue[[]*SignatureStatus]](ctx, c.Client, "getSignatureStatuses", sigStrs, FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return &GetSignatureStatusesResult{Slot: resp.Context.Slot, Statuses: resp.Value}, nil
}

// ConfirmedSignatureForAddress is a single entry in the result of GetSignaturesForAddress.
type ConfirmedSignatureForAddress struct {
	Signature          string  `json:"signature"`
	Slot               uint64  `json:"slot"`
	Err                any     `json:"err"`
	Memo               *string `json:"memo"`
	BlockTime          *int64  `json:"blockTime"`
	ConfirmationStatus string  `json:"confirmationStatus"`
}

// GetSignaturesForAddress fetches the transaction signatures that touched the given address.
func (c *Client) GetSignaturesForAddress(ctx context.Context, addr solana.PublicKey, cfg ...SignaturesForAddressCfg) ([]*ConfirmedSignatureForAddress, error) {
	result, err := jsonrpc.CallContext[[]*ConfirmedSignatureForAddress](ctx, c.Client, "getSignaturesForAddress", addr.String(), FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return result, nil
}
