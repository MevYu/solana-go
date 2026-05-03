package rpc

import (
	"context"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"
)

// EpochInfo is the decoded response of GetEpochInfo.
type EpochInfo struct {
	AbsoluteSlot     uint64  `json:"absoluteSlot"`
	BlockHeight      uint64  `json:"blockHeight"`
	Epoch            uint64  `json:"epoch"`
	SlotIndex        uint64  `json:"slotIndex"`
	SlotsInEpoch     uint64  `json:"slotsInEpoch"`
	TransactionCount *uint64 `json:"transactionCount"`
}

// GetEpochInfo returns information about the current epoch.
func (c *Client) GetEpochInfo(ctx context.Context, cfg ...CommitmentWithMinSlotCfg) (*EpochInfo, error) {
	result, err := jsonrpc.CallContext[EpochInfo](ctx, c.Client, "getEpochInfo", FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SupplyResult is the decoded response of GetSupply. Slot is filled
// from the JSON-RPC context envelope after decode; the rest of the
// fields decode directly via their json: tags.
type SupplyResult struct {
	Slot                   uint64   `json:"-"`
	Total                  uint64   `json:"total"`
	Circulating            uint64   `json:"circulating"`
	NonCirculating         uint64   `json:"nonCirculating"`
	NonCirculatingAccounts []string `json:"nonCirculatingAccounts"`
}

// GetSupply returns circulating and total SOL supply.
func (c *Client) GetSupply(ctx context.Context, cfg ...CommitmentCfg) (*SupplyResult, error) {
	resp, err := jsonrpc.CallContext[jsonrpc.ContextValue[SupplyResult]](ctx, c.Client, "getSupply", FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	resp.Value.Slot = resp.Context.Slot
	return &resp.Value, nil
}

// InflationRate is the decoded response of GetInflationRate.
type InflationRate struct {
	Total      float64 `json:"total"`
	Validator  float64 `json:"validator"`
	Foundation float64 `json:"foundation"`
	Epoch      uint64  `json:"epoch"`
}

// GetInflationRate returns the current inflation rate.
func (c *Client) GetInflationRate(ctx context.Context) (*InflationRate, error) {
	result, err := jsonrpc.CallContext[InflationRate](ctx, c.Client, "getInflationRate")
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetMinimumBalanceForRentExemption returns the minimum lamports needed to make an account rent-exempt.
func (c *Client) GetMinimumBalanceForRentExemption(ctx context.Context, dataSize uint64, cfg ...CommitmentCfg) (uint64, error) {
	lamports, err := jsonrpc.CallContext[uint64](ctx, c.Client, "getMinimumBalanceForRentExemption", dataSize, FirstOrZero(cfg))
	if err != nil {
		return 0, err
	}
	return lamports, nil
}

// GetSlotLeader returns the public key of the current slot leader.
func (c *Client) GetSlotLeader(ctx context.Context, cfg ...CommitmentWithMinSlotCfg) (solana.PublicKey, error) {
	leader, err := jsonrpc.CallContext[string](ctx, c.Client, "getSlotLeader", FirstOrZero(cfg))
	if err != nil {
		return solana.PublicKey{}, err
	}
	return solana.PublicKeyFromBase58(leader)
}

// GetBlockTime returns the UNIX timestamp at which the given slot was produced.
func (c *Client) GetBlockTime(ctx context.Context, slot uint64) (*int64, error) {
	ts, err := jsonrpc.CallContext[*int64](ctx, c.Client, "getBlockTime", slot)
	if err != nil {
		return nil, err
	}
	return ts, nil
}

// EpochSchedule holds the epoch schedule configuration of the cluster.
type EpochSchedule struct {
	SlotsPerEpoch            uint64 `json:"slotsPerEpoch"`
	LeaderScheduleSlotOffset uint64 `json:"leaderScheduleSlotOffset"`
	Warmup                   bool   `json:"warmup"`
	FirstNormalEpoch         uint64 `json:"firstNormalEpoch"`
	FirstNormalSlot          uint64 `json:"firstNormalSlot"`
}

// GetEpochSchedule returns the epoch schedule configuration from the cluster's genesis config.
func (c *Client) GetEpochSchedule(ctx context.Context) (*EpochSchedule, error) {
	result, err := jsonrpc.CallContext[EpochSchedule](ctx, c.Client, "getEpochSchedule")
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSlotLeaders returns the slot leaders for a range of slots.
func (c *Client) GetSlotLeaders(ctx context.Context, startSlot uint64, limit uint64) ([]solana.PublicKey, error) {
	raw, err := jsonrpc.CallContext[[]string](ctx, c.Client, "getSlotLeaders", startSlot, limit)
	if err != nil {
		return nil, err
	}
	leaders := make([]solana.PublicKey, len(raw))
	for i, s := range raw {
		pk, err := solana.PublicKeyFromBase58(s)
		if err != nil {
			return nil, err
		}
		leaders[i] = pk
	}
	return leaders, nil
}

// GetLeaderSchedule returns the leader schedule for an epoch. Pass nil
// for slot to query the current epoch.
func (c *Client) GetLeaderSchedule(ctx context.Context, slot *uint64, cfg ...LeaderScheduleCfg) (map[string][]uint64, error) {
	result, err := jsonrpc.CallContext[map[string][]uint64](ctx, c.Client, "getLeaderSchedule", slot, FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return result, nil
}

// InflationGovernor holds the inflation configuration of the cluster.
type InflationGovernor struct {
	Initial        float64 `json:"initial"`
	Terminal       float64 `json:"terminal"`
	Taper          float64 `json:"taper"`
	Foundation     float64 `json:"foundation"`
	FoundationTerm float64 `json:"foundationTerm"`
}

// GetInflationGovernor returns the current inflation governor configuration.
func (c *Client) GetInflationGovernor(ctx context.Context, cfg ...CommitmentCfg) (*InflationGovernor, error) {
	result, err := jsonrpc.CallContext[InflationGovernor](ctx, c.Client, "getInflationGovernor", FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// InflationReward is a single entry in the GetInflationReward response.
type InflationReward struct {
	Epoch         uint64 `json:"epoch"`
	EffectiveSlot uint64 `json:"effectiveSlot"`
	Amount        uint64 `json:"amount"`
	PostBalance   uint64 `json:"postBalance"`
	Commission    *uint8 `json:"commission"`
}

// GetInflationReward returns the inflation/staking reward for a list of addresses for an epoch.
// Set cfg.Epoch to query a specific epoch; leave nil for the previous epoch.
func (c *Client) GetInflationReward(ctx context.Context, addresses []solana.PublicKey, cfg ...InflationRewardCfg) ([]*InflationReward, error) {
	addrs := make([]string, len(addresses))
	for i, a := range addresses {
		addrs[i] = a.String()
	}
	result, err := jsonrpc.CallContext[[]*InflationReward](ctx, c.Client, "getInflationReward", addrs, FirstOrZero(cfg))
	if err != nil {
		return nil, err
	}
	return result, nil
}

// PerformanceSample is a single entry in the GetRecentPerformanceSamples response.
type PerformanceSample struct {
	Slot                   uint64 `json:"slot"`
	NumTransactions        uint64 `json:"numTransactions"`
	NumSlots               uint64 `json:"numSlots"`
	SamplePeriodSecs       uint16 `json:"samplePeriodSecs"`
	NumNonVoteTransactions uint64 `json:"numNonVoteTransactions"`
}

// GetRecentPerformanceSamples returns a list of recent performance samples.
func (c *Client) GetRecentPerformanceSamples(ctx context.Context, limit *uint64) ([]PerformanceSample, error) {
	result, err := jsonrpc.CallContext[[]PerformanceSample](ctx, c.Client, "getRecentPerformanceSamples", limit)
	if err != nil {
		return nil, err
	}
	return result, nil
}
