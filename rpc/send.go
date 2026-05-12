package rpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	solana "github.com/MevYu/solana-go"
	"github.com/MevYu/solana-go/jsonrpc"
)

// TransactionBuilder builds a fresh signed transaction with the given
// blockhash. SendAndConfirmTransaction calls it once per send attempt,
// so the builder must re-sign with the same signers for every call;
// captured signers and instructions are the expected pattern.
type TransactionBuilder func(ctx context.Context, blockhash solana.Hash) (*solana.Transaction, error)

// SendAndConfirmOption configures SendAndConfirmTransaction and
// SendAndConfirmSignedTransaction.
type SendAndConfirmOption func(*sendConfig)

type sendConfig struct {
	commitment          solana.CommitmentLevel
	confirmTimeout      time.Duration
	pollInterval        time.Duration
	maxBlockhashRetries int
	skipPreflight       *bool
}

func defaultSendConfig() sendConfig {
	return sendConfig{
		commitment:          solana.CommitmentConfirmed,
		confirmTimeout:      60 * time.Second,
		pollInterval:        2 * time.Second,
		maxBlockhashRetries: 3,
	}
}

// WithSendCommitment sets the commitment level required to consider a
// transaction confirmed. Default: Confirmed.
func WithSendCommitment(c solana.CommitmentLevel) SendAndConfirmOption {
	return func(cfg *sendConfig) { cfg.commitment = c }
}

// WithConfirmTimeout caps the total time to wait for confirmation.
// Default: 60 seconds.
func WithConfirmTimeout(d time.Duration) SendAndConfirmOption {
	return func(cfg *sendConfig) { cfg.confirmTimeout = d }
}

// WithPollInterval sets the delay between getSignatureStatuses polls
// while waiting for confirmation. Default: 2 seconds.
func WithPollInterval(d time.Duration) SendAndConfirmOption {
	return func(cfg *sendConfig) { cfg.pollInterval = d }
}

// WithMaxBlockhashRetries caps the number of blockhash refresh + rebuild
// + resend cycles SendAndConfirmTransaction will perform before giving
// up. Default: 3.
func WithMaxBlockhashRetries(n int) SendAndConfirmOption {
	return func(cfg *sendConfig) { cfg.maxBlockhashRetries = n }
}

// WithSendSkipPreflight disables the server's preflight check. Use only
// when the caller has already simulated the transaction and is sure it
// is valid.
func WithSendSkipPreflight(b bool) SendAndConfirmOption {
	return func(cfg *sendConfig) { cfg.skipPreflight = &b }
}

// errBlockhashExpired signals the confirmation loop that the
// transaction's blockhash has aged out and a refresh is required.
var errBlockhashExpired = errors.New("solana client: blockhash expired")

// SendAndConfirmTransaction builds, sends, and waits for a Solana
// transaction to reach the requested commitment level.
//
// The builder is called for each send attempt with the latest blockhash,
// so callers get automatic blockhash refresh: when a transaction's
// blockhash expires, SendAndConfirmTransaction fetches a fresh one,
// rebuilds via the builder, and resubmits — up to WithMaxBlockhashRetries
// times.
//
// On success it returns the first-signature of the transaction (the
// payer's). On preflight failure it returns the error from the send
// call. On execution failure (detected via getSignatureStatuses) it
// returns the decoded error from DecodeTransactionError, which
// unwraps to *TransactionError or *InstructionError.
func (c *Client) SendAndConfirmTransaction(
	ctx context.Context,
	build TransactionBuilder,
	opts ...SendAndConfirmOption,
) (solana.Signature, error) {
	if build == nil {
		return solana.Signature{}, errors.New("solana client: SendAndConfirmTransaction: nil builder")
	}
	cfg := defaultSendConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	var lastErr error
	for attempt := 0; attempt <= cfg.maxBlockhashRetries; attempt++ {
		bh, err := c.GetLatestBlockhash(ctx, CommitmentWithMinSlotCfg{Commitment: cfg.commitment})
		if err != nil {
			return solana.Signature{}, fmt.Errorf("solana client: get blockhash: %w", err)
		}

		tx, err := build(ctx, bh.Blockhash)
		if err != nil {
			return solana.Signature{}, fmt.Errorf("solana client: build transaction: %w", err)
		}

		sig, err := c.sendOnce(ctx, tx, cfg)
		if err != nil {
			if jsonrpc.IsBlockhashExpired(err) && attempt < cfg.maxBlockhashRetries {
				lastErr = err
				continue
			}
			return solana.Signature{}, err
		}

		if err := c.confirmSignature(ctx, sig, &bh.LastValidBlockHeight, cfg); err != nil {
			if errors.Is(err, errBlockhashExpired) && attempt < cfg.maxBlockhashRetries {
				lastErr = err
				continue
			}
			return sig, err
		}
		return sig, nil
	}
	if lastErr != nil {
		return solana.Signature{}, fmt.Errorf("solana client: exhausted blockhash retries: %w", lastErr)
	}
	return solana.Signature{}, errors.New("solana client: exhausted blockhash retries")
}

// SendAndConfirmSignedTransaction sends an already-signed transaction
// and waits for confirmation. Unlike SendAndConfirmTransaction, it
// performs exactly one send attempt and cannot refresh the blockhash on
// expiry; use it when the caller has a fully prepared transaction and
// wants a simple fire-and-wait flow.
func (c *Client) SendAndConfirmSignedTransaction(
	ctx context.Context,
	tx *solana.Transaction,
	opts ...SendAndConfirmOption,
) (solana.Signature, error) {
	if tx == nil {
		return solana.Signature{}, errors.New("solana client: SendAndConfirmSignedTransaction: nil transaction")
	}
	cfg := defaultSendConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	sig, err := c.sendOnce(ctx, tx, cfg)
	if err != nil {
		return solana.Signature{}, err
	}
	if err := c.confirmSignature(ctx, sig, nil, cfg); err != nil {
		return sig, err
	}
	return sig, nil
}

func (c *Client) sendOnce(ctx context.Context, tx *solana.Transaction, cfg sendConfig) (solana.Signature, error) {
	sendCfg := SendTxCfg{
		PreflightCommitment: cfg.commitment,
		SkipPreflight:       cfg.skipPreflight,
	}
	sig, err := c.SendTransaction(ctx, tx, sendCfg)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("solana client: send: %w", err)
	}
	return sig, nil
}

// confirmSignature polls getSignatureStatuses until the signature reaches
// the requested commitment level, the deadline fires, or the blockhash
// expires. lastValidBlockHeight is nil to disable the expiry check (the
// SendAndConfirmSignedTransaction path, where no blockhash was fetched).
func (c *Client) confirmSignature(
	ctx context.Context,
	sig solana.Signature,
	lastValidBlockHeight *uint64,
	cfg sendConfig,
) error {
	deadline := time.Now().Add(cfg.confirmTimeout)

	searchHistory := true
	statusCfg := SignatureStatusesCfg{SearchTransactionHistory: &searchHistory}
	heightCfg := CommitmentWithMinSlotCfg{Commitment: cfg.commitment}
	sigs := []solana.Signature{sig}

	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("solana client: confirmation timeout after %v", cfg.confirmTimeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var landed bool
		statuses, err := c.GetSignatureStatuses(ctx, sigs, statusCfg)
		if err == nil && len(statuses.Statuses) > 0 && statuses.Statuses[0] != nil {
			landed = true
			s := statuses.Statuses[0]
			// DecodeTransactionError treats a nil RawMessage, an empty
			// RawMessage, or a literal "null" payload as success.
			if txErr := DecodeTransactionError(s.Err); txErr != nil {
				return txErr
			}
			if statusReachedCommitment(s.ConfirmationStatus, cfg.commitment) {
				return nil
			}
		}

		// Only probe block height when the tx is not yet visible — once
		// the cluster has recorded the signature, expiry no longer matters.
		if !landed && lastValidBlockHeight != nil {
			if height, err := c.GetBlockHeight(ctx, heightCfg); err == nil {
				if height > *lastValidBlockHeight {
					return errBlockhashExpired
				}
			}
		}

		if timer == nil {
			timer = time.NewTimer(cfg.pollInterval)
		} else {
			timer.Reset(cfg.pollInterval)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func commitmentRank(c solana.CommitmentLevel) int {
	switch c {
	case solana.CommitmentProcessed:
		return 0
	case solana.CommitmentConfirmed:
		return 1
	case solana.CommitmentFinalized:
		return 2
	}
	return -1
}

func statusReachedCommitment(status string, required solana.CommitmentLevel) bool {
	statusN := commitmentRank(solana.CommitmentLevel(status))
	reqN := commitmentRank(required)
	if statusN < 0 || reqN < 0 {
		return false
	}
	return statusN >= reqN
}
