package rpc

import (
	"context"
	"encoding/base64"
	"fmt"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"

	"github.com/mr-tron/base58"
)

// GetTransactionResult is the decoded response of GetTransaction.
//
// Transaction is the fully decoded structured form: the JSON-RPC
// payload is `["<encoded-bytes>", "<encoding>"]`, and
// Transaction.UnmarshalJSON decodes those bytes through to a typed
// Transaction. Read tx.Message.Instructions etc. directly — no extra
// UnmarshalBinary step is needed. Only binary encodings (base64,
// base58, base64+zstd) are supported; json / jsonParsed return a
// nested object and are not handled.
type GetTransactionResult struct {
	Slot        uint64                  `json:"slot"`
	BlockTime   *int64                  `json:"blockTime"`
	Meta        *solana.TransactionMeta `json:"meta"`
	Transaction *solana.Transaction     `json:"transaction"`
	Version     any                     `json:"version"`
}

// GetTransaction fetches a single transaction by signature.
// Returns nil, nil if the signature is not known to the node.
// Encoding defaults to base64 and MaxSupportedTransactionVersion to 0
// when the caller leaves them unset.
func (c *Client) GetTransaction(ctx context.Context, sig solana.Signature, cfg ...GetTransactionCfg) (*GetTransactionResult, error) {
	c0 := *FirstOrZero(cfg)
	if c0.Encoding == "" {
		c0.Encoding = solana.EncodingBase64
	}
	if c0.MaxSupportedTransactionVersion == nil {
		zero := uint64(0)
		c0.MaxSupportedTransactionVersion = &zero
	}

	res, err := jsonrpc.CallContext[*GetTransactionResult](ctx, c.Client, "getTransaction", sig.String(), c0)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// SendTransaction marshals tx and broadcasts it to the cluster.
// The transaction must already be signed by every required signer.
func (c *Client) SendTransaction(ctx context.Context, tx *solana.Transaction, cfg ...SendTxCfg) (solana.Signature, error) {
	raw, err := tx.Marshal()
	if err != nil {
		return solana.Signature{}, fmt.Errorf("solana: SendTransaction: marshal: %w", err)
	}
	return c.SendRawTransaction(ctx, raw, cfg...)
}

// SendRawTransaction broadcasts pre-marshaled transaction bytes to the cluster.
// Encoding may be base58 or base64; defaults to base64.
func (c *Client) SendRawTransaction(ctx context.Context, raw []byte, cfg ...SendTxCfg) (solana.Signature, error) {
	c0 := *FirstOrZero(cfg)
	if c0.Encoding == "" {
		c0.Encoding = solana.EncodingBase64
	}

	var encoded string
	switch c0.Encoding {
	case solana.EncodingBase64:
		encoded = base64.StdEncoding.EncodeToString(raw)
	case solana.EncodingBase58:
		encoded = base58.Encode(raw)
	default:
		return solana.Signature{}, fmt.Errorf("solana: SendRawTransaction: unsupported encoding %q", c0.Encoding)
	}

	sigStr, err := jsonrpc.CallContext[string](ctx, c.Client, "sendTransaction", encoded, c0)
	if err != nil {
		return solana.Signature{}, err
	}
	return solana.SignatureFromBase58(sigStr)
}
