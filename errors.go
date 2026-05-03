package solana

import "errors"

// ErrInvalidLength is returned when a byte slice or a base58-decoded
// value has the wrong size for the target Solana primitive (public
// key, hash, signature, ...). Callers can detect it with errors.Is
// instead of string matching.
var ErrInvalidLength = errors.New("solana: invalid length")

// ErrInvalidBase58 is returned when a string cannot be decoded as
// base58 because it is empty or contains characters outside the
// base58 alphabet.
var ErrInvalidBase58 = errors.New("solana: invalid base58")

