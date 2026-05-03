package rpc

import (
	"context"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"
)

// ProgramAccount is a single entry in the GetProgramAccounts response.
type ProgramAccount struct {
	Pubkey  solana.PublicKey   `json:"pubkey"`
	Account solana.AccountInfo `json:"account"`
}

// GetProgramAccounts returns all accounts owned by the given program.
// Encoding defaults to base64.
func (c *Client) GetProgramAccounts(ctx context.Context, program solana.PublicKey, cfg ...AccountInfoCfg) ([]ProgramAccount, error) {
	result, err := jsonrpc.CallContext[[]ProgramAccount](ctx, c.Client, "getProgramAccounts", program.String(), FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return result, nil
}

// TokenAccountFilter selects token accounts by either mint or token program.
type TokenAccountFilter struct {
	Mint      solana.PublicKey
	ProgramID solana.PublicKey
}

// TokenAccount is a single entry in the GetTokenAccountsByOwner and GetTokenAccountsByDelegate responses.
type TokenAccount struct {
	Pubkey  solana.PublicKey   `json:"pubkey"`
	Account solana.AccountInfo `json:"account"`
}

// GetTokenAccountsByOwner returns all SPL Token accounts owned by the given wallet address.
// Encoding defaults to base64.
func (c *Client) GetTokenAccountsByOwner(ctx context.Context, owner solana.PublicKey, filter TokenAccountFilter, cfg ...AccountInfoCfg) ([]TokenAccount, error) {
	return c.tokenAccountsCall(ctx, "getTokenAccountsByOwner", owner, filter, cfg)
}

// GetTokenAccountsByDelegate returns all SPL Token accounts for which the given address has been approved as a delegate.
// Encoding defaults to base64.
func (c *Client) GetTokenAccountsByDelegate(ctx context.Context, delegate solana.PublicKey, filter TokenAccountFilter, cfg ...AccountInfoCfg) ([]TokenAccount, error) {
	return c.tokenAccountsCall(ctx, "getTokenAccountsByDelegate", delegate, filter, cfg)
}

func (c *Client) tokenAccountsCall(ctx context.Context, method string, addr solana.PublicKey, filter TokenAccountFilter, cfg []AccountInfoCfg) ([]TokenAccount, error) {
	// Filter is one of {mint:..} or {programId:..}; pick whichever is set.
	var f map[string]string
	if !filter.Mint.IsZero() {
		f = map[string]string{"mint": filter.Mint.String()}
	} else {
		f = map[string]string{"programId": filter.ProgramID.String()}
	}
	resp, err := jsonrpc.CallContext[jsonrpc.ContextValue[[]TokenAccount]](ctx, c.Client, method, addr.String(), f, FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return resp.Value, nil
}

// TokenLargestAccount is a single entry in the GetTokenLargestAccounts response.
type TokenLargestAccount struct {
	Address        solana.PublicKey `json:"address"`
	Amount         string           `json:"amount"`
	Decimals       uint8            `json:"decimals"`
	UIAmount       *float64         `json:"uiAmount"`
	UIAmountString string           `json:"uiAmountString"`
}

// GetTokenLargestAccounts returns the 20 largest accounts for a given SPL Token mint.
func (c *Client) GetTokenLargestAccounts(ctx context.Context, mint solana.PublicKey, cfg ...CommitmentCfg) ([]TokenLargestAccount, error) {
	resp, err := jsonrpc.CallContext[jsonrpc.ContextValue[[]TokenLargestAccount]](ctx, c.Client, "getTokenLargestAccounts", mint.String(), FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return resp.Value, nil
}

// LargestAccount is a single entry in the GetLargestAccounts response.
type LargestAccount struct {
	Address  solana.PublicKey `json:"address"`
	Lamports uint64           `json:"lamports"`
}

// GetLargestAccounts returns the 20 largest accounts by lamport balance.
// cfg.Filter selects "circulating" or "nonCirculating".
func (c *Client) GetLargestAccounts(ctx context.Context, cfg ...LargestAccountsCfg) ([]LargestAccount, error) {
	resp, err := jsonrpc.CallContext[jsonrpc.ContextValue[[]LargestAccount]](ctx, c.Client, "getLargestAccounts", FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return resp.Value, nil
}
