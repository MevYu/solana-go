package encoding

// Chainable writer methods on Encoder.
//
// These short, chainable aliases sit on top of the existing Write* methods
// and let callers build wire-format payloads top-to-bottom in a single
// expression — useful for instruction-data builders and similar small
// fixed-shape structures:
//
//	data := encoding.NewEncoder(48).
//	    Discriminator(swapDisc).
//	    U64(amountIn).
//	    U64(amountOutMin).
//	    U128(sqrtPxLimit).
//	    Raw(userPubkey[:]).
//	    Bool(aToB).
//	    Bytes()
//
// The package stays free of Solana high-level types (PublicKey, Hash, …):
// callers pass slices via Raw, matching the convention used by the legacy
// encodbin / bin-go libraries.

// ─── unsigned ────────────────────────────────────────────────────────────────

// U8 appends a single byte and returns e for chaining.
func (e *Encoder) U8(v uint8) *Encoder { e.WriteUint8(v); return e }

// U16 appends a little-endian uint16 and returns e for chaining.
func (e *Encoder) U16(v uint16) *Encoder { e.WriteUint16(v); return e }

// U32 appends a little-endian uint32 and returns e for chaining.
func (e *Encoder) U32(v uint32) *Encoder { e.WriteUint32(v); return e }

// U64 appends a little-endian uint64 and returns e for chaining.
func (e *Encoder) U64(v uint64) *Encoder { e.WriteUint64(v); return e }

// ─── signed ──────────────────────────────────────────────────────────────────

// I8 appends a signed int8 and returns e for chaining.
func (e *Encoder) I8(v int8) *Encoder { e.WriteInt8(v); return e }

// I16 appends a little-endian int16 and returns e for chaining.
func (e *Encoder) I16(v int16) *Encoder { e.WriteInt16(v); return e }

// I32 appends a little-endian int32 and returns e for chaining.
func (e *Encoder) I32(v int32) *Encoder { e.WriteInt32(v); return e }

// I64 appends a little-endian int64 and returns e for chaining.
func (e *Encoder) I64(v int64) *Encoder { e.WriteInt64(v); return e }

// ─── 128 / 256-bit ───────────────────────────────────────────────────────────

// U128c is the chainable form of WriteU128. (U128 is taken by the type name.)
func (e *Encoder) U128c(v U128) *Encoder { e.WriteU128(v); return e }

// U256c is the chainable form of WriteU256.
func (e *Encoder) U256c(v U256) *Encoder { e.WriteU256(v); return e }

// ─── misc ────────────────────────────────────────────────────────────────────

// Bool writes 1 byte: 1 for true, 0 for false (Rust bincode bool encoding).
func (e *Encoder) Bool(v bool) *Encoder {
	if v {
		e.WriteUint8(1)
	} else {
		e.WriteUint8(0)
	}
	return e
}

// Raw appends b verbatim (no length prefix). Use for fixed-size byte arrays
// such as pubkeys (caller passes pk[:]) or pre-encoded discriminators.
func (e *Encoder) Raw(b []byte) *Encoder { e.WriteBytes(b); return e }

// Discriminator is an 8-byte Anchor instruction discriminator. Identical to
// Raw(d[:]) but the named method documents intent at the call site.
func (e *Encoder) Discriminator(d [8]byte) *Encoder { e.WriteBytes(d[:]); return e }

// Str writes a bincode string: u64 little-endian length, then UTF-8 bytes.
func (e *Encoder) Str(s string) *Encoder {
	e.WriteUint64(uint64(len(s)))
	e.WriteBytes([]byte(s))
	return e
}

// Shortvec writes a Solana compact-u16 (1-3 bytes). Used inside transaction
// and message structures, not in instruction data.
func (e *Encoder) Shortvec(v uint16) *Encoder { e.WriteShortvec(v); return e }

// ─── Rust Option<T> ──────────────────────────────────────────────────────────
//
// All Opt* methods write a 1-byte discriminant (0=None, 1=Some) followed by
// the payload when non-nil, matching Rust bincode's Option<T> encoding.

// OptU8 writes Option<u8>.
func (e *Encoder) OptU8(v *uint8) *Encoder {
	if v == nil {
		return e.U8(0)
	}
	return e.U8(1).U8(*v)
}

// OptU32 writes Option<u32>.
func (e *Encoder) OptU32(v *uint32) *Encoder {
	if v == nil {
		return e.U8(0)
	}
	return e.U8(1).U32(*v)
}

// OptU64 writes Option<u64>.
func (e *Encoder) OptU64(v *uint64) *Encoder {
	if v == nil {
		return e.U8(0)
	}
	return e.U8(1).U64(*v)
}

// OptI64 writes Option<i64>.
func (e *Encoder) OptI64(v *int64) *Encoder {
	if v == nil {
		return e.U8(0)
	}
	return e.U8(1).I64(*v)
}

// OptU128 writes Option<u128>.
func (e *Encoder) OptU128(v *U128) *Encoder {
	if v == nil {
		return e.U8(0)
	}
	return e.U8(1).U128c(*v)
}

// OptBool writes Option<bool>.
func (e *Encoder) OptBool(v *bool) *Encoder {
	if v == nil {
		return e.U8(0)
	}
	return e.U8(1).Bool(*v)
}

// OptRaw writes a Rust Option<[N]byte>: 1-byte tag, then b verbatim
// when non-nil. Pass nil to encode None. The slice length is not
// length-prefixed: this is intended for fixed-size optional fields
// such as Option<Pubkey> via OptRaw(pk[:]).
func (e *Encoder) OptRaw(b []byte) *Encoder {
	if b == nil {
		return e.U8(0)
	}
	return e.U8(1).Raw(b)
}
