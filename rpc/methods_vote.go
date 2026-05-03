package rpc

import (
	"context"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"
)

// VoteAccount holds the state of a single vote account.
type VoteAccount struct {
	VotePubkey       solana.PublicKey `json:"votePubkey"`
	NodePubkey       solana.PublicKey `json:"nodePubkey"`
	ActivatedStake   uint64           `json:"activatedStake"`
	EpochVoteAccount bool             `json:"epochVoteAccount"`
	Commission       uint8            `json:"commission"`
	LastVote         uint64           `json:"lastVote"`
	EpochCredits     [][3]uint64      `json:"epochCredits"`
	RootSlot         uint64           `json:"rootSlot"`
}

// VoteAccounts is the decoded response of GetVoteAccounts.
type VoteAccounts struct {
	Current    []VoteAccount `json:"current"`
	Delinquent []VoteAccount `json:"delinquent"`
}

// GetVoteAccounts returns the account info and associated stake for all voting accounts.
func (c *Client) GetVoteAccounts(ctx context.Context, cfg ...CommitmentCfg) (*VoteAccounts, error) {
	result, err := jsonrpc.CallContext[VoteAccounts](ctx, c.Client, "getVoteAccounts", FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ClusterNode describes a single node in the cluster.
type ClusterNode struct {
	Pubkey       solana.PublicKey `json:"pubkey"`
	Gossip       string           `json:"gossip"`
	TPU          string           `json:"tpu"`
	RPC          string           `json:"rpc"`
	Version      string           `json:"version"`
	FeatureSet   *uint32          `json:"featureSet"`
	ShredVersion *uint16          `json:"shredVersion"`
}

// GetClusterNodes returns information about all nodes participating in the cluster.
func (c *Client) GetClusterNodes(ctx context.Context) ([]ClusterNode, error) {
	result, err := jsonrpc.CallContext[[]ClusterNode](ctx, c.Client, "getClusterNodes")
	if err != nil {
		return nil, err
	}
	return result, nil
}

// BlockProductionRange describes the slot range covered by a GetBlockProduction response.
type BlockProductionRange struct {
	FirstSlot uint64 `json:"firstSlot"`
	LastSlot  uint64 `json:"lastSlot"`
}

// BlockProductionValue is the data portion of a GetBlockProduction response.
type BlockProductionValue struct {
	ByIdentity map[string][2]uint64 `json:"byIdentity"`
	Range      BlockProductionRange `json:"range"`
}

// BlockProductionResult is the decoded response of GetBlockProduction.
type BlockProductionResult struct {
	Slot  uint64
	Value BlockProductionValue
}

// GetBlockProduction returns recent block production information.
func (c *Client) GetBlockProduction(ctx context.Context, cfg ...CommitmentCfg) (*BlockProductionResult, error) {
	resp, err := jsonrpc.CallContext[jsonrpc.ContextValue[BlockProductionValue]](ctx, c.Client, "getBlockProduction", FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return &BlockProductionResult{Slot: resp.Context.Slot, Value: resp.Value}, nil
}

// StakeActivation is the decoded response of GetStakeActivation.
type StakeActivation struct {
	State    string `json:"state"`
	Active   uint64 `json:"active"`
	Inactive uint64 `json:"inactive"`
}

// GetStakeActivation returns the activation state of a stake account.
func (c *Client) GetStakeActivation(ctx context.Context, account solana.PublicKey, cfg ...CommitmentWithMinSlotCfg) (*StakeActivation, error) {
	result, err := jsonrpc.CallContext[StakeActivation](ctx, c.Client, "getStakeActivation", account.String(), FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetStakeMinimumDelegation returns the stake minimum delegation in lamports.
func (c *Client) GetStakeMinimumDelegation(ctx context.Context, cfg ...CommitmentCfg) (uint64, error) {
	resp, err := jsonrpc.CallContext[jsonrpc.ContextValue[uint64]](ctx, c.Client, "getStakeMinimumDelegation", FirstOrZero(cfg))
	if err != nil {
		return 0, err
	}
	return resp.Value, nil
}
