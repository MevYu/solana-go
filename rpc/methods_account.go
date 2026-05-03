package rpc

import (
	"context"
	"fmt"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"
)

// GetAccountInfoResult is the decoded response of GetAccountInfo.
type GetAccountInfoResult struct {
	Slot    uint64
	Account *solana.AccountInfo
}

// GetAccountInfo fetches the current state of the account at the given address.
// Encoding defaults to base64 when cfg.Encoding is empty.
func (c *Client) GetAccountInfo(ctx context.Context, pubkey solana.PublicKey, cfg ...AccountInfoCfg) (*GetAccountInfoResult, error) {
	resp, err := jsonrpc.CallContext[jsonrpc.ContextValue[*solana.AccountInfo]](ctx, c.Client, "getAccountInfo", pubkey.String(), FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return &GetAccountInfoResult{Slot: resp.Context.Slot, Account: resp.Value}, nil
}

// GetMultipleAccountsResult is the decoded response of GetMultipleAccounts.
type GetMultipleAccountsResult struct {
	Slot     uint64
	Accounts []*solana.AccountInfo
}

// MaxGetMultipleAccountsAddresses is the per-request limit the Solana
// RPC server enforces on getMultipleAccounts. The SDK validates input
// length up-front so callers get a precise error instead of a vague
// "Invalid params" from the server.
const MaxGetMultipleAccountsAddresses = 100

// GetMultipleAccounts fetches multiple accounts in a single round trip.
// Encoding defaults to base64 when cfg.Encoding is empty.
func (c *Client) GetMultipleAccounts(ctx context.Context, addresses []solana.PublicKey, cfg ...AccountInfoCfg) (*GetMultipleAccountsResult, error) {
	if len(addresses) > MaxGetMultipleAccountsAddresses {
		return nil, fmt.Errorf("solana: GetMultipleAccounts: %d addresses exceeds Solana RPC max of %d", len(addresses), MaxGetMultipleAccountsAddresses)
	}
	keys := make([]string, len(addresses))
	for i, a := range addresses {
		keys[i] = a.String()
	}
	resp, err := jsonrpc.CallContext[jsonrpc.ContextValue[[]*solana.AccountInfo]](ctx, c.Client, "getMultipleAccounts", keys, FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return &GetMultipleAccountsResult{Slot: resp.Context.Slot, Accounts: resp.Value}, nil
}
