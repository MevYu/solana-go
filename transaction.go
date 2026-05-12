package solana

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"

	"github.com/MevYu/solana-go/encoding"

	"github.com/mr-tron/base58"
)

// Transaction is a Solana transaction: a message together with one
// signature per required signer. The wire encoding is a shortvec
// length prefix, then each signature as 64 raw bytes, then the
// serialized message body.
//
// A freshly constructed Transaction has zero-filled signature slots
// for each required signer. Calling Sign fills in the slots for the
// signers that were passed, leaving all others untouched. This
// supports incremental signing: a caller may collect signatures from
// multiple parties by calling Sign more than once with different
// signer subsets.
type Transaction struct {
	// Signatures holds one entry per required signer, ordered to
	// match the first Message.Header.NumRequiredSignatures entries
	// of Message.AccountKeys. Unfilled slots contain the all-zero
	// Signature.
	Signatures []Signature

	// Message is the transaction body.
	Message Message
}

// NewTransaction constructs an unsigned transaction for message.
// The Signatures slice is pre-allocated with zero-filled placeholder
// entries for each required signer, matching the header count.
func NewTransaction(message Message) *Transaction {
	return &Transaction{
		Signatures: make([]Signature, message.Header.NumRequiredSignatures),
		Message:    message,
	}
}

// SerializedSize returns the wire-format byte count for this signed
// transaction without allocating the encoded buffer. The result equals
// len(tx.Marshal()) for any valid transaction.
func (tx *Transaction) SerializedSize() int {
	n := len(tx.Signatures)
	// shortvec for n: 1 byte when n < 128 (always true for ≤35 signers)
	svLen := 1
	if n >= 128 {
		svLen = 2
	}
	return svLen + n*SignatureSize + tx.Message.SerializedSize()
}

// MessageBytes returns the serialized message body that signers must
// sign over. It is equivalent to Message.Marshal and is exposed as a
// convenience for offline signing workflows.
func (tx *Transaction) MessageBytes() ([]byte, error) {
	return tx.Message.Marshal()
}

// Sign signs the transaction with the provided signers. Every signer
// must own a public key that appears in the first
// Message.Header.NumRequiredSignatures entries of
// Message.AccountKeys; otherwise Sign returns an error naming the
// offending signer and does not mutate any state.
//
// Signature slots are filled in-place at the index matching each
// signer's position in the key list. The lookup from public key to
// slot is O(1): a map is built once per call, so the total cost
// scales linearly with the number of signers rather than with
// signers times required slots, satisfying the project's
// architectural principles on lookup scaling.
//
// The serialized message body is computed once and reused across
// signers, so marshalling cost is amortised even when many signers
// are passed in a single call.
//
// After Sign returns, any slot whose signer did not appear in the
// signers list retains its previous value (zero for a fresh
// transaction). Callers may call Sign multiple times with different
// signer sets to accumulate signatures over time, for example when
// coordinating a multi-party signing flow.
//
// ctx is forwarded to every signer's Sign method, so a cancellation
// or deadline on ctx will reach remote signers mid-flight.
func (tx *Transaction) Sign(ctx context.Context, signers ...Signer) error {
	numReq := int(tx.Message.Header.NumRequiredSignatures)
	if numReq == 0 {
		return fmt.Errorf("solana: transaction: message requires zero signatures")
	}
	if len(tx.Message.AccountKeys) < numReq {
		return fmt.Errorf("solana: transaction: message has %d account keys but header requires %d signatures",
			len(tx.Message.AccountKeys), numReq)
	}
	if len(tx.Signatures) < numReq {
		// Grow to hold every required slot, preserving any existing
		// signatures that might already be present.
		grown := make([]Signature, numReq)
		copy(grown, tx.Signatures)
		tx.Signatures = grown
	}

	// Validate the signers list before committing any mutations: if
	// any signer is not a required signer, we abort without having
	// touched tx.Signatures.
	index := make(map[PublicKey]int, numReq)
	for i := 0; i < numReq; i++ {
		index[tx.Message.AccountKeys[i]] = i
	}
	slots := make([]int, len(signers))
	for i, s := range signers {
		slot, ok := index[s.PublicKey()]
		if !ok {
			return fmt.Errorf("solana: transaction: signer %s is not a required signer", s.PublicKey())
		}
		slots[i] = slot
	}

	// Marshal the message once and reuse it across all signers.
	msgBytes, err := tx.Message.Marshal()
	if err != nil {
		return fmt.Errorf("solana: transaction: marshal message: %w", err)
	}

	for i, s := range signers {
		sig, err := s.Sign(ctx, msgBytes)
		if err != nil {
			return fmt.Errorf("solana: transaction: signer %s: %w", s.PublicKey(), err)
		}
		tx.Signatures[slots[i]] = sig
	}

	return nil
}

// Marshal returns the wire-format encoding of the transaction. The
// returned slice is newly allocated and owned by the caller.
func (tx *Transaction) Marshal() ([]byte, error) {
	if len(tx.Signatures) > 0xFFFF {
		return nil, fmt.Errorf("solana: transaction: too many signatures: %d", len(tx.Signatures))
	}
	if err := tx.Message.validate(); err != nil {
		return nil, err
	}
	// Refuse to encode a transaction whose signature slot count does not
	// match Message.Header.NumRequiredSignatures: validators reject any
	// such payload, and emitting it produces silently malformed bytes
	// the caller cannot easily diagnose.
	if got, want := len(tx.Signatures), int(tx.Message.Header.NumRequiredSignatures); got != want {
		return nil, fmt.Errorf("solana: transaction: %d signatures for %d required signers", got, want)
	}
	e := encoding.NewEncoder(tx.SerializedSize())
	e.WriteShortvec(uint16(len(tx.Signatures)))
	for i := range tx.Signatures {
		e.WriteBytes(tx.Signatures[i][:])
	}
	if err := tx.Message.marshalInto(e); err != nil {
		return nil, err
	}
	return append([]byte(nil), e.Bytes()...), nil
}

// UnmarshalBinary parses a wire-format Solana transaction into tx,
// implementing encoding.BinaryUnmarshaler. Decoded fields do not
// alias data: signatures are copied out, and the embedded Message is
// fully materialised by DecodeMessage, which itself copies its
// variable-length fields.
//
// Trailing bytes after the transaction are rejected. To decode a
// transaction out of a larger stream, use DecodeTransaction with a
// caller-owned *encoding.Decoder.
func (tx *Transaction) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("solana: transaction: empty input")
	}
	d := encoding.NewDecoder(data)
	decoded, err := DecodeTransaction(d)
	if err != nil {
		return err
	}
	if r := d.Remaining(); r != 0 {
		return fmt.Errorf("solana: transaction: %d trailing bytes", r)
	}
	*tx = *decoded
	return nil
}

// DecodeTransaction reads one Solana Transaction from d's current
// position and advances the cursor past the message body. Unlike
// Transaction.UnmarshalBinary it does NOT enforce end-of-buffer, so
// callers can decode a sequence of transactions from a larger stream
// (e.g. Jito ShredStream entries).
//
// The returned Transaction does not alias d's buffer.
func DecodeTransaction(d *encoding.Decoder) (*Transaction, error) {
	sigCount, err := d.ReadShortvec()
	if err != nil {
		return nil, fmt.Errorf("solana: transaction: signatures count: %w", err)
	}
	sigs := make([]Signature, sigCount)
	for i := uint16(0); i < sigCount; i++ {
		b, err := d.ReadBytes(SignatureSize)
		if err != nil {
			return nil, fmt.Errorf("solana: transaction: signature %d: %w", i, err)
		}
		copy(sigs[i][:], b)
	}

	msg, err := DecodeMessage(d)
	if err != nil {
		return nil, fmt.Errorf("solana: transaction: message: %w", err)
	}
	return &Transaction{Signatures: sigs, Message: *msg}, nil
}

// UnmarshalFromDecoder implements encoding.Unmarshaler so that Transaction
// can be decoded generically via d.DecodeTo(&tx) from within a larger stream.
func (tx *Transaction) UnmarshalFromDecoder(d *encoding.Decoder) error {
	decoded, err := DecodeTransaction(d)
	if err != nil {
		return err
	}
	*tx = *decoded
	return nil
}

// NewTransactionFromInstructions is a go-solana-compatible convenience
// that builds a Message from instructions, recent blockhash, and payer,
// then wraps it into an unsigned Transaction. It mirrors the go-solana
// NewTransaction(instructions, recentBlockHash, payer) signature.
func NewTransactionFromInstructions(instructions []Instruction, recentBlockhash Hash, payer PublicKey) (*Transaction, error) {
	msg, err := NewMessage(payer, instructions, recentBlockhash)
	if err != nil {
		return nil, err
	}
	return NewTransaction(*msg), nil
}

// MarshalBinary is an alias for Marshal, matching the go-solana
// naming convention.
func (tx *Transaction) MarshalBinary() ([]byte, error) {
	return tx.Marshal()
}

// ToBase64 returns the base64-encoded wire format of the transaction.
func (tx *Transaction) ToBase64() (string, error) {
	data, err := tx.Marshal()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// ToBase58 returns the base58-encoded wire format of the transaction.
func (tx *Transaction) ToBase58() (string, error) {
	data, err := tx.Marshal()
	if err != nil {
		return "", err
	}
	return base58.Encode(data), nil
}

// VerifySignatures checks every filled (non-zero) signature against
// the current message bytes using stdlib Ed25519 verification. A zero
// signature slot is treated as unsigned and is skipped. This is a
// best-effort local sanity check and is not a substitute for the
// validator's full transaction-validity rules.
func (tx *Transaction) VerifySignatures() error {
	numReq := int(tx.Message.Header.NumRequiredSignatures)
	if len(tx.Signatures) < numReq {
		return fmt.Errorf("solana: transaction: only %d signature slots for %d required signers",
			len(tx.Signatures), numReq)
	}
	if len(tx.Message.AccountKeys) < numReq {
		return fmt.Errorf("solana: transaction: only %d account keys for %d required signers",
			len(tx.Message.AccountKeys), numReq)
	}
	msgBytes, err := tx.Message.Marshal()
	if err != nil {
		return err
	}
	for i := 0; i < numReq; i++ {
		if tx.Signatures[i].IsZero() {
			continue
		}
		pub := tx.Message.AccountKeys[i]
		if !ed25519.Verify(ed25519.PublicKey(pub[:]), msgBytes, tx.Signatures[i][:]) {
			return fmt.Errorf("solana: transaction: signature at slot %d for %s does not verify", i, pub)
		}
	}
	return nil
}
