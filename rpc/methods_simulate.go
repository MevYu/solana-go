package rpc

import (
	"context"
	"encoding/base64"
	"fmt"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"
)

// SimulateTransaction simulates the given transaction without broadcasting it.
// Encoding defaults to base64 when cfg.Encoding is empty.
func (c *Client) SimulateTransaction(ctx context.Context, tx *solana.Transaction, cfg ...SimulateTxCfg) (*SimulateResult, error) {
	raw, err := tx.Marshal()
	if err != nil {
		return nil, fmt.Errorf("solana: SimulateTransaction: marshal: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(raw)

	c0 := *FirstOrZero(cfg)
	if c0.Encoding == "" {
		c0.Encoding = solana.EncodingBase64
	}

	var resp jsonrpc.ContextValue[SimulateResult]
	if err := c.CallContext(ctx, &resp, "simulateTransaction", encoded, c0); err != nil {
		return nil, err
	}
	resp.Value.Slot = resp.Context.Slot
	return &resp.Value, nil
}
