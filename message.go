package solana

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/MevYu/solana-go/encoding"
)

// MessageVersion identifies the wire-format version of a Message.
//
// On-wire, the version is the message's first byte: a legacy message
// uses MessageVersionLegacy (0xFF, an in-memory sentinel that is
// never written to the wire), and a versioned message stores
// `versionPrefixMask | version` as that byte.
//
// JSON-RPC responses (getTransaction, getBlock, blockSubscribe) emit
// the version as either the string "legacy" or the integer version
// number; MessageVersion's UnmarshalJSON / MarshalJSON convert both
// directions, so callers can use a single typed Version field across
// the wire-format and the RPC envelope.
type MessageVersion uint8

const (
	// MessageVersionLegacy is the pre-v0 Solana message format. A
	// legacy message has no version prefix byte on the wire: its
	// serialized form begins directly with MessageHeader. This value
	// is a Go-level sentinel only; it is never written to the wire.
	MessageVersionLegacy MessageVersion = 0xFF

	// MessageVersion0 is the first versioned message format, which
	// introduces Address Lookup Table support. On the wire, a v0
	// message begins with the byte versionPrefixMask | 0 == 0x80.
	MessageVersion0 MessageVersion = 0
)

// UnmarshalJSON decodes the JSON-RPC representation of a transaction
// version. Accepted forms:
//
//   - JSON null, empty string, or "legacy" → MessageVersionLegacy
//   - any unsigned integer N → MessageVersion(N)
//
// The decoder does not enforce the supported-version range — that
// check happens in the wire-format Marshal / DecodeMessage path. A
// JSON response carrying an unknown integer version is accepted here
// and only rejected when the caller tries to act on it.
func (v *MessageVersion) UnmarshalJSON(data []byte) error {
	s := string(data)
	switch s {
	case "null", `""`, `"legacy"`:
		*v = MessageVersionLegacy
		return nil
	}
	n, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return fmt.Errorf("solana: MessageVersion: %w", err)
	}
	*v = MessageVersion(n)
	return nil
}

// MarshalJSON emits the JSON-RPC representation: "legacy" for the
// legacy sentinel, otherwise the integer version number.
func (v MessageVersion) MarshalJSON() ([]byte, error) {
	if v == MessageVersionLegacy {
		return []byte(`"legacy"`), nil
	}
	return []byte(strconv.FormatUint(uint64(v), 10)), nil
}

// versionPrefixMask is the high bit of the first byte that
// distinguishes a versioned message from a legacy one.
//
// A legacy message's first byte is numRequiredSignatures, which is
// strictly less than 0x80 in any valid transaction (Solana caps it
// well under 128 signers per transaction). A versioned message sets
// this bit and uses the remaining 7 bits to encode the version number.
// We use the explicit mask rather than an ad-hoc comparison so the
// convention is named and testable.
const versionPrefixMask byte = 0x80

// maxMessageVersion is the highest versioned message format this
// package understands. Bump when the next version ships upstream.
const maxMessageVersion MessageVersion = 0

// MessageHeader describes the signing and writability layout of a
// Message's static account keys.
type MessageHeader struct {
	// NumRequiredSignatures is the total number of accounts that must
	// sign the transaction.
	NumRequiredSignatures uint8

	// NumReadonlySignedAccounts is how many of the first
	// NumRequiredSignatures accounts are read-only.
	NumReadonlySignedAccounts uint8

	// NumReadonlyUnsignedAccounts is how many of the remaining
	// unsigned accounts are read-only.
	NumReadonlyUnsignedAccounts uint8
}

// CompiledInstruction is an Instruction whose program and accounts
// have been resolved to indices into the enclosing Message's account
// key array.
type CompiledInstruction struct {
	// ProgramIDIndex is the position of the program account in the
	// message's AccountKeys.
	ProgramIDIndex uint8

	// Accounts holds the indices of the instruction's input accounts
	// within the message's AccountKeys; for v0 messages indices may
	// point past the static keys into addresses resolved through
	// AddressTableLookups.
	//
	// Typed as Uint8Slice (not []uint8) so encoding/json renders it as
	// a JSON array of numbers, matching the Solana JSON-RPC shape,
	// instead of as a base64 string.
	Accounts Uint8Slice

	// Data is the serialized instruction payload. On the JSON-RPC wire
	// it is a base58 string (getTransaction/getBlock meta inner
	// instructions), so it uses Base58Data rather than a plain []byte,
	// which encoding/json would (incorrectly) treat as base64.
	Data Base58Data
}

// MessageAddressTableLookup is a v0-only reference to an Address
// Lookup Table that supplies additional accounts beyond the static
// keys in a Message.
type MessageAddressTableLookup struct {
	// AccountKey is the address of the Address Lookup Table account.
	AccountKey PublicKey

	// WritableIndexes are the indices within the lookup table of
	// accounts that instructions in this message may write to. Typed
	// as Uint8Slice so JSON rendering is a number array.
	WritableIndexes Uint8Slice

	// ReadonlyIndexes are the indices within the lookup table of
	// accounts that are strictly read-only in this message. Typed as
	// Uint8Slice so JSON rendering is a number array.
	ReadonlyIndexes Uint8Slice
}

// Uint8Slice is a slice of uint8s that can be marshaled as numbers instead of a byte slice.
type Uint8Slice []uint8

// MarshalJSON implements json.Marshaler.
func (u8s Uint8Slice) MarshalJSON() ([]byte, error) {
	out := make([]uint16, len(u8s))
	for i, idx := range u8s {
		out[i] = uint16(idx)
	}
	return json.Marshal(out)
}

// Message is the serialized body of a Solana transaction. It supports
// both the legacy format (Version == MessageVersionLegacy) and the
// versioned format (Version == MessageVersion0).
type Message struct {
	// Version is the wire-format version. Only MessageVersionLegacy
	// and MessageVersion0 are currently supported.
	Version MessageVersion

	// Header counts the signing and read-only static accounts.
	Header MessageHeader

	// AccountKeys is the static account array. Legacy messages use
	// only this list; v0 messages extend it with accounts resolved
	// through AddressTableLookups.
	AccountKeys []PublicKey

	// RecentBlockhash is the blockhash the transaction commits to.
	RecentBlockhash Hash

	// Instructions is the ordered list of compiled instructions.
	Instructions []CompiledInstruction

	// AddressTableLookups is empty for legacy messages and may be
	// non-empty for v0 messages.
	AddressTableLookups []MessageAddressTableLookup
}

// MarshalBinary is an alias for Marshal, matching the go-solana
// naming convention.
func (m *Message) MarshalBinary() ([]byte, error) {
	return m.Marshal()
}

// svSize returns the number of bytes the shortvec (compact-u16) encoding
// of n occupies on the wire: 1 byte for n<128, 2 for n<16384, else 3.
func svSize(n int) int {
	switch {
	case n < 128:
		return 1
	case n < 16384:
		return 2
	default:
		return 3
	}
}

// SerializedSize returns the exact wire-format byte count for this
// message without allocating the encoded buffer. The result equals
// len(m.Marshal()) for any valid message.
func (m *Message) SerializedSize() int {
	sz := 3 // header bytes
	if m.Version != MessageVersionLegacy {
		sz++ // version prefix byte
	}
	sz += svSize(len(m.AccountKeys)) + len(m.AccountKeys)*PublicKeySize
	sz += HashSize
	sz += svSize(len(m.Instructions))
	for i := range m.Instructions {
		ix := &m.Instructions[i]
		sz += 1 + svSize(len(ix.Accounts)) + len(ix.Accounts) + svSize(len(ix.Data)) + len(ix.Data)
	}
	if m.Version == MessageVersion0 {
		sz += svSize(len(m.AddressTableLookups))
		for i := range m.AddressTableLookups {
			lk := &m.AddressTableLookups[i]
			sz += PublicKeySize + svSize(len(lk.WritableIndexes)) + len(lk.WritableIndexes) + svSize(len(lk.ReadonlyIndexes)) + len(lk.ReadonlyIndexes)
		}
	}
	return sz
}

// Marshal returns the wire-format encoding of the message. The
// returned slice is newly allocated and owned by the caller.
func (m *Message) Marshal() ([]byte, error) {
	if err := m.validate(); err != nil {
		return nil, err
	}
	e := encoding.NewEncoder(m.SerializedSize())
	if err := m.marshalInto(e); err != nil {
		return nil, err
	}
	// e is a fresh local sized by SerializedSize and not retained, so its
	// buffer is already a caller-owned slice — no extra copy needed.
	return e.Bytes(), nil
}

// marshalInto writes the wire-format message body into e. The caller is
// responsible for validating the message before calling this method.
func (m *Message) marshalInto(e *encoding.Encoder) error {
	// 1. Version prefix (absent for legacy).
	if m.Version != MessageVersionLegacy {
		e.WriteUint8(versionPrefixMask | byte(m.Version))
	}

	// 2. Header.
	e.WriteUint8(m.Header.NumRequiredSignatures)
	e.WriteUint8(m.Header.NumReadonlySignedAccounts)
	e.WriteUint8(m.Header.NumReadonlyUnsignedAccounts)

	// 3. Static account keys.
	if len(m.AccountKeys) > 0xFFFF {
		return fmt.Errorf("solana: message: marshal: too many account keys (%d, max %d)", len(m.AccountKeys), 0xFFFF)
	}
	e.WriteShortvec(uint16(len(m.AccountKeys)))
	for i := range m.AccountKeys {
		e.WriteBytes(m.AccountKeys[i][:])
	}

	// 4. Recent blockhash.
	e.WriteBytes(m.RecentBlockhash[:])

	// 5. Compiled instructions.
	if len(m.Instructions) > 0xFFFF {
		return fmt.Errorf("solana: message: marshal: too many instructions (%d, max %d)", len(m.Instructions), 0xFFFF)
	}
	e.WriteShortvec(uint16(len(m.Instructions)))
	for i := range m.Instructions {
		ix := &m.Instructions[i]
		e.WriteUint8(ix.ProgramIDIndex)

		if len(ix.Accounts) > 0xFFFF {
			return fmt.Errorf("solana: message: marshal: instruction %d has too many account indices (%d, max %d)", i, len(ix.Accounts), 0xFFFF)
		}
		e.WriteShortvec(uint16(len(ix.Accounts)))
		e.WriteBytes(ix.Accounts)

		if len(ix.Data) > 0xFFFF {
			return fmt.Errorf("solana: message: marshal: instruction %d data too long (%d, max %d)", i, len(ix.Data), 0xFFFF)
		}
		e.WriteShortvec(uint16(len(ix.Data)))
		e.WriteBytes(ix.Data)
	}

	// 6. Address table lookups (v0 only).
	if m.Version == MessageVersion0 {
		if len(m.AddressTableLookups) > 0xFFFF {
			return fmt.Errorf("solana: message: marshal: too many address table lookups (%d, max %d)", len(m.AddressTableLookups), 0xFFFF)
		}
		e.WriteShortvec(uint16(len(m.AddressTableLookups)))
		for i := range m.AddressTableLookups {
			lk := &m.AddressTableLookups[i]
			e.WriteBytes(lk.AccountKey[:])

			if len(lk.WritableIndexes) > 0xFFFF {
				return fmt.Errorf("solana: message: marshal: lookup %d has too many writable indices (%d, max %d)", i, len(lk.WritableIndexes), 0xFFFF)
			}
			e.WriteShortvec(uint16(len(lk.WritableIndexes)))
			e.WriteBytes(lk.WritableIndexes)

			if len(lk.ReadonlyIndexes) > 0xFFFF {
				return fmt.Errorf("solana: message: marshal: lookup %d has too many readonly indices (%d, max %d)", i, len(lk.ReadonlyIndexes), 0xFFFF)
			}
			e.WriteShortvec(uint16(len(lk.ReadonlyIndexes)))
			e.WriteBytes(lk.ReadonlyIndexes)
		}
	}
	return nil
}

// Signers returns the static account keys that must sign this message.
// Signers are always static (the runtime verifies signatures before
// resolving address-table lookups), so this is identical for legacy and
// v0 messages.
func (m *Message) Signers() []PublicKey {
	// Clamp to len(AccountKeys): a message decoded from untrusted bytes
	// can carry NumRequiredSignatures > len(AccountKeys), which would
	// otherwise panic on the slice expression.
	n := int(m.Header.NumRequiredSignatures)
	if n > len(m.AccountKeys) {
		n = len(m.AccountKeys)
	}
	return m.AccountKeys[:n]
}

// ResolvedAccountKeys flattens a v0 message's index space into a single
// []PublicKey slice that callers can index with raw instruction account
// indices. alts maps each AddressTableLookup.AccountKey to that lookup
// table's full address list (TableState.Addresses); the caller is
// responsible for fetching ALT state via rpc.Client.GetAddressLookupTable
// or GetMultipleAccounts before invoking this.
//
// Index layout (Solana protocol order):
//
//	[0, len(AccountKeys))                       static keys
//	[..., +Σ lookups[i].WritableIndexes)        ALT writable, in AddressTableLookups order
//	[..., +Σ lookups[i].ReadonlyIndexes)        ALT readonly, in AddressTableLookups order
//
// For legacy messages (AddressTableLookups empty) alts is unused and
// m.AccountKeys is returned directly without copying. To resolve a
// specific instruction's accounts, call this once per message and index
// resolved[ix.Accounts[i]].
func (m *Message) ResolvedAccountKeys(alts map[PublicKey][]PublicKey) ([]PublicKey, error) {
	if len(m.AddressTableLookups) == 0 {
		return m.AccountKeys, nil
	}
	total := len(m.AccountKeys)
	for i := range m.AddressTableLookups {
		total += len(m.AddressTableLookups[i].WritableIndexes) + len(m.AddressTableLookups[i].ReadonlyIndexes)
	}
	out := make([]PublicKey, 0, total)
	out = append(out, m.AccountKeys...)
	for i := range m.AddressTableLookups {
		lk := &m.AddressTableLookups[i]
		addrs, ok := alts[lk.AccountKey]
		if !ok {
			return nil, fmt.Errorf("solana: message: ALT %s not in alts map", lk.AccountKey)
		}
		for _, idx := range lk.WritableIndexes {
			if int(idx) >= len(addrs) {
				return nil, fmt.Errorf("solana: message: ALT %s writable index %d out of range (table has %d entries)", lk.AccountKey, idx, len(addrs))
			}
			out = append(out, addrs[idx])
		}
	}
	for i := range m.AddressTableLookups {
		lk := &m.AddressTableLookups[i]
		addrs := alts[lk.AccountKey] // presence already validated above
		for _, idx := range lk.ReadonlyIndexes {
			if int(idx) >= len(addrs) {
				return nil, fmt.Errorf("solana: message: ALT %s readonly index %d out of range (table has %d entries)", lk.AccountKey, idx, len(addrs))
			}
			out = append(out, addrs[idx])
		}
	}
	return out, nil
}

func (m *Message) validate() error {
	switch m.Version {
	case MessageVersionLegacy:
		if len(m.AddressTableLookups) > 0 {
			return fmt.Errorf("solana: message: legacy messages cannot carry address table lookups")
		}
	case MessageVersion0:
		// supported
	default:
		return fmt.Errorf("solana: message: unsupported version %d", m.Version)
	}
	return nil
}

// UnmarshalJSON decodes the [value, encoding] tuple form Solana
// JSON-RPC returns for raw message bytes, then parses the decoded
// bytes into a typed Message. See Transaction.UnmarshalJSON for the
// same pattern at the transaction level.
func (m *Message) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var d EncodedData
	if err := json.Unmarshal(data, &d); err != nil {
		return fmt.Errorf("solana: Message: %w", err)
	}
	if len(d.Bytes) == 0 {
		return nil
	}
	return m.UnmarshalBinary(d.Bytes)
}

// UnmarshalBinary parses data as a Solana Message and populates m,
// implementing encoding.BinaryUnmarshaler. The decoded fields do not
// alias data: all variable-length payloads are copied out of the
// decoder before UnmarshalBinary returns, so callers may mutate or
// free data afterwards.
//
// Trailing bytes after the message body are rejected. To decode a
// message out of a larger stream, use DecodeMessage with a
// caller-owned *encoding.Decoder.
func (m *Message) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("solana: message: empty input")
	}
	d := encoding.NewDecoder(data)
	decoded, err := DecodeMessage(d)
	if err != nil {
		return err
	}
	if r := d.Remaining(); r != 0 {
		return fmt.Errorf("solana: message: %d trailing bytes after message body", r)
	}
	*m = *decoded
	return nil
}

// DecodeMessage reads a Solana Message from d's current position and
// advances the cursor past the last field. Unlike Message.UnmarshalBinary
// it does NOT enforce end-of-buffer, so callers can decode a message
// out of a larger byte stream (e.g. Jito ShredStream entries).
//
// The returned Message does not alias d's buffer: all variable-length
// fields are copied out before returning.
func DecodeMessage(d *encoding.Decoder) (*Message, error) {
	m := &Message{}

	// 1. Version prefix.
	first, err := d.PeekUint8()
	if err != nil {
		return nil, fmt.Errorf("solana: message: version byte: %w", err)
	}
	if first&versionPrefixMask != 0 {
		b, _ := d.ReadUint8()
		v := MessageVersion(b &^ versionPrefixMask)
		if v > maxMessageVersion {
			return nil, fmt.Errorf("solana: message: unsupported version %d", v)
		}
		m.Version = v
	} else {
		m.Version = MessageVersionLegacy
	}

	// 2. Header.
	h1, err := d.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("solana: message: header numRequiredSignatures: %w", err)
	}
	h2, err := d.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("solana: message: header numReadonlySignedAccounts: %w", err)
	}
	h3, err := d.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("solana: message: header numReadonlyUnsignedAccounts: %w", err)
	}
	m.Header = MessageHeader{
		NumRequiredSignatures:       h1,
		NumReadonlySignedAccounts:   h2,
		NumReadonlyUnsignedAccounts: h3,
	}

	// 3. Static account keys.
	keyCount, err := d.ReadShortvec()
	if err != nil {
		return nil, fmt.Errorf("solana: message: account keys count: %w", err)
	}
	m.AccountKeys = make([]PublicKey, keyCount)
	for i := uint16(0); i < keyCount; i++ {
		kb, err := d.ReadBytes(PublicKeySize)
		if err != nil {
			return nil, fmt.Errorf("solana: message: account key %d: %w", i, err)
		}
		copy(m.AccountKeys[i][:], kb)
	}

	// 4. Recent blockhash.
	bh, err := d.ReadBytes(HashSize)
	if err != nil {
		return nil, fmt.Errorf("solana: message: recent blockhash: %w", err)
	}
	copy(m.RecentBlockhash[:], bh)

	// 5. Compiled instructions.
	ixCount, err := d.ReadShortvec()
	if err != nil {
		return nil, fmt.Errorf("solana: message: instruction count: %w", err)
	}
	m.Instructions = make([]CompiledInstruction, ixCount)
	for i := uint16(0); i < ixCount; i++ {
		pid, err := d.ReadUint8()
		if err != nil {
			return nil, fmt.Errorf("solana: message: instruction %d program index: %w", i, err)
		}

		ac, err := d.ReadShortvec()
		if err != nil {
			return nil, fmt.Errorf("solana: message: instruction %d accounts count: %w", i, err)
		}
		accAlias, err := d.ReadBytes(int(ac))
		if err != nil {
			return nil, fmt.Errorf("solana: message: instruction %d accounts: %w", i, err)
		}
		accounts := make(Uint8Slice, ac)
		copy(accounts, accAlias)

		dl, err := d.ReadShortvec()
		if err != nil {
			return nil, fmt.Errorf("solana: message: instruction %d data count: %w", i, err)
		}
		dataAlias, err := d.ReadBytes(int(dl))
		if err != nil {
			return nil, fmt.Errorf("solana: message: instruction %d data: %w", i, err)
		}
		dataBytes := make([]byte, dl)
		copy(dataBytes, dataAlias)

		m.Instructions[i] = CompiledInstruction{
			ProgramIDIndex: pid,
			Accounts:       accounts,
			Data:           dataBytes,
		}
	}

	// 6. Address table lookups (v0 only).
	if m.Version == MessageVersion0 {
		lkCount, err := d.ReadShortvec()
		if err != nil {
			return nil, fmt.Errorf("solana: message: address table lookups count: %w", err)
		}
		m.AddressTableLookups = make([]MessageAddressTableLookup, lkCount)
		for i := uint16(0); i < lkCount; i++ {
			lk := &m.AddressTableLookups[i]

			kb, err := d.ReadBytes(PublicKeySize)
			if err != nil {
				return nil, fmt.Errorf("solana: message: lookup %d account key: %w", i, err)
			}
			copy(lk.AccountKey[:], kb)

			wc, err := d.ReadShortvec()
			if err != nil {
				return nil, fmt.Errorf("solana: message: lookup %d writable count: %w", i, err)
			}
			wAlias, err := d.ReadBytes(int(wc))
			if err != nil {
				return nil, fmt.Errorf("solana: message: lookup %d writable: %w", i, err)
			}
			lk.WritableIndexes = make(Uint8Slice, wc)
			copy(lk.WritableIndexes, wAlias)

			rc, err := d.ReadShortvec()
			if err != nil {
				return nil, fmt.Errorf("solana: message: lookup %d readonly count: %w", i, err)
			}
			rAlias, err := d.ReadBytes(int(rc))
			if err != nil {
				return nil, fmt.Errorf("solana: message: lookup %d readonly: %w", i, err)
			}
			lk.ReadonlyIndexes = make(Uint8Slice, rc)
			copy(lk.ReadonlyIndexes, rAlias)
		}
	}

	return m, nil
}
