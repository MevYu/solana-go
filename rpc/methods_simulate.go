package rpc

import (
	"context"
	"encoding/base64"
	"fmt"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"
	"github.com/mr-tron/base58"
)

// SimulateTransaction simulates the given transaction without broadcasting it.
// Encoding defaults to base64 when cfg.Encoding is empty.
func (c *Client) SimulateTransaction(ctx context.Context, tx *solana.Transaction, cfg ...SimulateTxCfg) (*SimulateResult, error) {
	if tx == nil {
		return nil, fmt.Errorf("solana: SimulateTransaction: nil transaction")
	}

	raw, err := tx.Marshal()
	if err != nil {
		return nil, fmt.Errorf("solana: SimulateTransaction: marshal: %w", err)
	}

	c0 := *FirstOrZero(cfg)
	if c0.Encoding == "" {
		c0.Encoding = solana.EncodingBase64
	}
	if c0.SigVerify != nil && *c0.SigVerify &&
		c0.ReplaceRecentBlockhash != nil && *c0.ReplaceRecentBlockhash {
		return nil, fmt.Errorf("solana: SimulateTransaction: sigVerify and replaceRecentBlockhash cannot both be true")
	}

	if c0.Accounts != nil {
		accounts := *c0.Accounts
		if accounts.Encoding == "" {
			accounts.Encoding = solana.EncodingBase64
		}
		switch accounts.Encoding {
		case solana.EncodingBase64, solana.EncodingBase64ZSTD:
			// simulateTransaction rejects base58 account snapshots; these are
			// the raw encodings supported by both the RPC and AccountInfo.
		default:
			return nil, fmt.Errorf("solana: SimulateTransaction: unsupported account encoding %q", accounts.Encoding)
		}
		c0.Accounts = &accounts
	}

	var encoded string
	switch c0.Encoding {
	case solana.EncodingBase64:
		encoded = base64.StdEncoding.EncodeToString(raw)
	case solana.EncodingBase58:
		encoded = base58.Encode(raw)
	default:
		return nil, fmt.Errorf("solana: SimulateTransaction: unsupported transaction encoding %q", c0.Encoding)
	}

	resp, err := jsonrpc.CallContext[jsonrpc.ContextValue[SimulateResult]](ctx, c.Client, "simulateTransaction", encoded, c0)
	if err != nil {
		return nil, err
	}
	resp.Value.Slot = resp.Context.Slot
	return &resp.Value, nil
}
