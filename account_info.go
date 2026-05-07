package solana

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/mr-tron/base58"
)

var (
	zstdDecOnce sync.Once
	zstdDec     *zstd.Decoder
	zstdEncOnce sync.Once
	zstdEnc     *zstd.Encoder
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

// globalZstdEncoder returns the process-wide zstd encoder used by
// EncodedData.MarshalJSON. klauspost zstd's Encoder.EncodeAll is safe for
// concurrent use after construction.
func globalZstdEncoder() *zstd.Encoder {
	zstdEncOnce.Do(func() {
		var err error
		zstdEnc, err = zstd.NewWriter(nil)
		if err != nil {
			panic("solana: failed to initialise zstd encoder: " + err.Error())
		}
	})
	return zstdEnc
}

// AccountInfo is the JSON representation of an account as returned
// by the getAccountInfo and getMultipleAccounts RPC methods.
//
// It differs from the wire-level Account type in that it preserves
// the RPC protocol's encoded data form, plus the server-reported
// Space field which is absent from the binary wire format.
type AccountInfo struct {
	Lamports   uint64      `json:"lamports"`
	Owner      PublicKey   `json:"owner"`
	Data       EncodedData `json:"data"`
	Executable bool        `json:"executable"`
	RentEpoch  uint64      `json:"rentEpoch"`
	Space      uint64      `json:"space"`
}

// EncodedData carries any binary payload the Solana JSON-RPC returns as a
// two-element [value, encoding] tuple — account data, simulated-instruction
// return values, and transaction wire bytes all share this shape.
//
// UnmarshalJSON eagerly decodes Value into Bytes according to Encoding;
// callers read Bytes directly and never see the on-wire string. MarshalJSON
// re-encodes Bytes back to the [value, encoding] tuple.
type EncodedData struct {
	Bytes    []byte
	Encoding Encoding
}

// UnmarshalJSON decodes a [value, encoding] JSON tuple. It returns an error
// if the array shape is wrong, the value cannot be decoded under the named
// encoding, or the encoding is jsonParsed/json (which return an object, not
// raw bytes — those callers must use a different type).
func (d *EncodedData) UnmarshalJSON(data []byte) error {
	var arr [2]string
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("solana: EncodedData: %w", err)
	}
	value, enc := arr[0], Encoding(arr[1])
	d.Encoding = enc
	if value == "" {
		d.Bytes = nil
		return nil
	}
	switch enc {
	case EncodingBase64:
		b, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return fmt.Errorf("solana: EncodedData: base64 decode: %w", err)
		}
		d.Bytes = b
	case EncodingBase58:
		b, err := base58.Decode(value)
		if err != nil {
			return fmt.Errorf("solana: EncodedData: base58 decode: %w", err)
		}
		d.Bytes = b
	case EncodingBase64ZSTD:
		compressed, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return fmt.Errorf("solana: EncodedData: base64 decode: %w", err)
		}
		out, err := globalZstdDecoder().DecodeAll(compressed, nil)
		if err != nil {
			return fmt.Errorf("solana: EncodedData: zstd decompress: %w", err)
		}
		d.Bytes = out
	case EncodingJSONParsed, EncodingJSON:
		return fmt.Errorf("solana: EncodedData: encoding %q is a parsed object, not raw bytes", enc)
	default:
		return fmt.Errorf("solana: EncodedData: unknown encoding %q", enc)
	}
	return nil
}

// MarshalJSON re-encodes Bytes under Encoding and emits the [value, encoding]
// tuple. jsonParsed/json encodings have no raw-bytes wire form and return an
// error.
func (d EncodedData) MarshalJSON() ([]byte, error) {
	var value string
	switch d.Encoding {
	case EncodingBase64:
		value = base64.StdEncoding.EncodeToString(d.Bytes)
	case EncodingBase58:
		value = base58.Encode(d.Bytes)
	case EncodingBase64ZSTD:
		compressed := globalZstdEncoder().EncodeAll(d.Bytes, nil)
		value = base64.StdEncoding.EncodeToString(compressed)
	case EncodingJSONParsed, EncodingJSON:
		return nil, fmt.Errorf("solana: EncodedData: encoding %q has no raw-bytes form", d.Encoding)
	case "":
		return nil, fmt.Errorf("solana: EncodedData: empty Encoding")
	default:
		return nil, fmt.Errorf("solana: EncodedData: unknown encoding %q", d.Encoding)
	}
	return json.Marshal([2]string{value, string(d.Encoding)})
}

// ToAccount converts an AccountInfo into the binary-compatible
// Account type. Use this when you want to store account state in
// the same shape the binary wire format uses, or when you plan to
// feed the account into program-specific binary decoders.
func (a *AccountInfo) ToAccount() *Account {
	return &Account{
		Lamports:   a.Lamports,
		Owner:      a.Owner,
		Data:       a.Data.Bytes,
		Executable: a.Executable,
		RentEpoch:  a.RentEpoch,
	}
}
