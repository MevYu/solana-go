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
	var result EpochInfo
	if err := c.CallContext(ctx, &result, "getEpochInfo", FirstOrZero(cfg)); err != nil {
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
	var resp jsonrpc.ContextValue[SupplyResult]
	if err := c.CallContext(ctx, &resp, "getSupply", FirstOrZero(cfg)); err != nil {
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
	var result InflationRate
	if err := c.CallContext(ctx, &result, "getInflationRate"); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetMinimumBalanceForRentExemption returns the minimum lamports needed to make an account rent-exempt.
func (c *Client) GetMinimumBalanceForRentExemption(ctx context.Context, dataSize uint64, cfg ...CommitmentCfg) (uint64, error) {
	var lamports uint64
	if err := c.CallContext(ctx, &lamports, "getMinimumBalanceForRentExemption", dataSize, FirstOrZero(cfg)); err != nil {
		return 0, err
	}
	return lamports, nil
}

// GetSlotLeader returns the public key of the current slot leader.
func (c *Client) GetSlotLeader(ctx context.Context, cfg ...CommitmentWithMinSlotCfg) (solana.PublicKey, error) {
	var leader string
	if err := c.CallContext(ctx, &leader, "getSlotLeader", FirstOrZero(cfg)); err != nil {
		return solana.PublicKey{}, err
	}
	return solana.PublicKeyFromBase58(leader)
}

// GetBlockTime returns the UNIX timestamp at which the given slot was produced.
func (c *Client) GetBlockTime(ctx context.Context, slot uint64) (*int64, error) {
	var ts *int64
	if err := c.CallContext(ctx, &ts, "getBlockTime", slot); err != nil {
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
	var result EpochSchedule
	if err := c.CallContext(ctx, &result, "getEpochSchedule"); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSlotLeaders returns the slot leaders for a range of slots.
func (c *Client) GetSlotLeaders(ctx context.Context, startSlot uint64, limit uint64) ([]solana.PublicKey, error) {
	var raw []string
	if err := c.CallContext(ctx, &raw, "getSlotLeaders", startSlot, limit); err != nil {
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
	var result map[string][]uint64
	if err := c.CallContext(ctx, &result, "getLeaderSchedule", slot, FirstOrZero(cfg)); err != nil {
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
	var result InflationGovernor
	if err := c.CallContext(ctx, &result, "getInflationGovernor", FirstOrZero(cfg)); err != nil {
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
	var result []*InflationReward
	if err := c.CallContext(ctx, &result, "getInflationReward", addrs, FirstOrZero(cfg)); err != nil {
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
	var result []PerformanceSample
	if err := c.CallContext(ctx, &result, "getRecentPerformanceSamples", limit); err != nil {
		return nil, err
	}
	return result, nil
}
