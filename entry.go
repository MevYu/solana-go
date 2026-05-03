package solana

import (
	"fmt"

	"github.com/MevYu/solana-go/encoding"
)

// Practical upper bounds for wire-format entry decoding. These values
// are generous multiples of the current Solana mainnet limits and exist
// solely to prevent OOM from a maliciously crafted buffer that encodes
// a huge count before any actual data.
const (
	maxEntriesPerDecode = 1_000_000 // entries per Vec<Entry> buffer
	maxTxPerEntry       = 10_000    // transactions per single entry
)

// Entry is a Solana Proof-of-History entry as emitted on the wire by
// Jito ShredStream (and by solana-core internally). An entry carries a
// chain link (NumHashes, Hash) and a batch of transactions that were
// sequenced at this point in the PoH stream.
//
// The bincode wire layout is:
//
//	num_hashes:   u64 little-endian
//	hash:         [32]byte
//	transactions: Vec<Transaction>   // u64 length prefix, then N wire-format txs
type Entry struct {
	NumHashes    uint64
	Hash         Hash
	Transactions []Transaction
}

// DecodeEntries parses a bincode Vec<Entry> buffer, as delivered by
// Jito ShredStream's Entry.Entries field. The outer Vec length is a
// little-endian u64 (bincode default), each entry has a u64 num_hashes
// and a 32-byte hash, followed by a u64-prefixed Vec<Transaction> in
// Solana wire format.
//
// The returned slice does not alias data: transaction signatures and
// message fields are copied out of the buffer.
func DecodeEntries(data []byte) ([]Entry, error) {
	d := encoding.AcquireDecoder(data)
	defer encoding.ReleaseDecoder(d)
	entryCount, err := d.ReadUint64()
	if err != nil {
		return nil, fmt.Errorf("solana: entries: count: %w", err)
	}
	if entryCount > maxEntriesPerDecode {
		return nil, fmt.Errorf("solana: entries: count %d exceeds limit %d", entryCount, maxEntriesPerDecode)
	}
	entries := make([]Entry, entryCount)
	for i := uint64(0); i < entryCount; i++ {
		if err := decodeEntryInto(d, &entries[i]); err != nil {
			return nil, fmt.Errorf("solana: entries: entry %d: %w", i, err)
		}
	}
	if r := d.Remaining(); r != 0 {
		return nil, fmt.Errorf("solana: entries: %d trailing bytes", r)
	}
	return entries, nil
}

// DecodeEntry reads one Entry from d's current position.
func DecodeEntry(d *encoding.Decoder) (*Entry, error) {
	e := &Entry{}
	if err := decodeEntryInto(d, e); err != nil {
		return nil, err
	}
	return e, nil
}

// UnmarshalFromDecoder implements encoding.Unmarshaler so that Entry can be
// decoded generically via d.DecodeTo(&entry).
func (e *Entry) UnmarshalFromDecoder(d *encoding.Decoder) error {
	return decodeEntryInto(d, e)
}

func decodeEntryInto(d *encoding.Decoder, e *Entry) error {
	n, err := d.ReadUint64()
	if err != nil {
		return fmt.Errorf("num_hashes: %w", err)
	}
	e.NumHashes = n

	hb, err := d.ReadBytes(HashSize)
	if err != nil {
		return fmt.Errorf("hash: %w", err)
	}
	copy(e.Hash[:], hb)

	txCount, err := d.ReadUint64()
	if err != nil {
		return fmt.Errorf("tx count: %w", err)
	}
	if txCount > maxTxPerEntry {
		return fmt.Errorf("tx count %d exceeds limit %d", txCount, maxTxPerEntry)
	}
	e.Transactions = make([]Transaction, txCount)
	for j := uint64(0); j < txCount; j++ {
		tx, err := DecodeTransaction(d)
		if err != nil {
			return fmt.Errorf("tx %d: %w", j, err)
		}
		e.Transactions[j] = *tx
	}
	return nil
}
