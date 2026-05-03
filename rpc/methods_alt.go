package rpc

import (
	"context"
	"fmt"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"
	"github.com/MevYu/solana-go/programs/address-lookup-table"
)

// GetAddressLookupTableResult is the decoded response of GetAddressLookupTable.
type GetAddressLookupTableResult struct {
	Slot  uint64
	State *addresslookuptable.TableState
}

// GetAddressLookupTable fetches and decodes the on-chain state of an
// Address Lookup Table at the given address. Returns nil, nil if the
// account does not exist.
//
// Encoding is forced to base64 — the table layout is binary and cannot
// be requested via jsonParsed. Honoured options: Commitment, MinContextSlot.
func (c *Client) GetAddressLookupTable(ctx context.Context, address solana.PublicKey, cfg ...CommitmentWithMinSlotCfg) (*GetAddressLookupTableResult, error) {
	opt := FirstOrZero(cfg)
	// Reuse AccountInfoCfg here (same wire shape) with encoding forced to
	// base64 — the table layout is binary.
	infoCfg := AccountInfoCfg{
		Commitment:     opt.Commitment,
		MinContextSlot: opt.MinContextSlot,
		Encoding:       solana.EncodingBase64,
	}
	var resp jsonrpc.ContextValue[*solana.AccountInfo]
	if err := c.CallContext(ctx, &resp, "getAccountInfo", address.String(), infoCfg); err != nil {
		return nil, err
	}
	slot, acc := resp.Context.Slot, resp.Value
	if acc == nil {
		return nil, nil
	}
	raw, err := acc.Data.Bytes()
	if err != nil {
		return nil, fmt.Errorf("client: GetAddressLookupTable: decode data: %w", err)
	}
	state, err := addresslookuptable.DecodeTableState(raw)
	if err != nil {
		return nil, fmt.Errorf("client: GetAddressLookupTable: %w", err)
	}
	return &GetAddressLookupTableResult{Slot: slot, State: state}, nil
}
