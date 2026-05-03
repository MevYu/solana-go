package solana

import (
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/mr-tron/base58"
)

var (
	zstdDecOnce sync.Once
	zstdDec     *zstd.Decoder
)

// globalZstdDecoder returns the process-wide zstd decoder, initialising it on
// first call. The decoder is safe for concurrent use by multiple goroutines.
func globalZstdDecoder() *zstd.Decoder {
	zstdDecOnce.Do(func() {
		var err error
		// WithDecoderConcurrency(0) → use GOMAXPROCS goroutines for parallel blocks.
		// WithDecoderLowmem(false) → keep speed tables in memory (default).
		zstdDec, err = zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
		if err != nil {
			panic("solana: failed to initialise zstd decoder: " + err.Error())
		}
	})
	return zstdDec
}

// AccountInfo is the JSON representation of an account as returned
// by the getAccountInfo and getMultipleAccounts RPC methods.
//
// It differs from the wire-level Account type in that it preserves
// the RPC protocol's two-element [value, encoding] data form, plus
// the server-reported Space field which is absent from the binary
// wire format.
type AccountInfo struct {
	Lamports   uint64      `json:"lamports"`
	Owner      PublicKey   `json:"owner"`
	Data       AccountData `json:"data"`
	Executable bool        `json:"executable"`
	RentEpoch  uint64      `json:"rentEpoch"`
	Space      uint64      `json:"space"`
}

// AccountData is the raw data field of an AccountInfo. On the wire
// Solana RPC returns it as a two-element JSON array:
//
//	["<encoded_value>", "<encoding_name>"]
//
// It is modelled as a fixed-size [2]string so both stdlib
// encoding/json and goccy/go-json decode it directly without a
// custom UnmarshalJSON. Use Value, Encoding, and Bytes to access
// it.
type AccountData [2]string

// Value returns the raw encoded value string.
func (d AccountData) Value() string { return d[0] }

// Encoding returns the wire encoding name.
func (d AccountData) Encoding() Encoding { return Encoding(d[1]) }

// Bytes decodes Value according to Encoding and returns the raw
// bytes. An empty Value returns a nil slice and no error. An
// unsupported encoding (jsonParsed, json, base64+zstd) returns an
// error describing the shortfall so the caller can decide how to
// handle it.
func (d AccountData) Bytes() ([]byte, error) {
	if d[0] == "" {
		return nil, nil
	}
	switch Encoding(d[1]) {
	case EncodingBase64:
		return base64.StdEncoding.DecodeString(d[0])
	case EncodingBase58:
		return base58.Decode(d[0])
	case EncodingBase64ZSTD:
		compressed, err := base64.StdEncoding.DecodeString(d[0])
		if err != nil {
			return nil, fmt.Errorf("solana: AccountData: base64 decode: %w", err)
		}
		out, err := globalZstdDecoder().DecodeAll(compressed, nil)
		if err != nil {
			return nil, fmt.Errorf("solana: AccountData: zstd decompress: %w", err)
		}
		return out, nil
	case EncodingJSONParsed, EncodingJSON:
		return nil, fmt.Errorf("solana: AccountData: encoding %q is a parsed object, not raw bytes", d[1])
	default:
		return nil, fmt.Errorf("solana: AccountData: unknown encoding %q", d[1])
	}
}

// ToAccount converts an AccountInfo into the binary-compatible
// Account type. It decodes the data field via Bytes so errors
// there propagate. Use this when you want to store account state
// in the same shape the binary wire format uses, or when you plan
// to feed the account into program-specific binary decoders.
func (a *AccountInfo) ToAccount() (*Account, error) {
	data, err := a.Data.Bytes()
	if err != nil {
		return nil, err
	}
	return &Account{
		Lamports:   a.Lamports,
		Owner:      a.Owner,
		Data:       data,
		Executable: a.Executable,
		RentEpoch:  a.RentEpoch,
	}, nil
}
