package solana

import (
	"fmt"
	"sort"
)

// LoadedAddressLookupTable holds a resolved Address Lookup Table: its
// on-chain key and the full ordered list of addresses stored in the table
// (slot 0, 1, 2, …). Obtain this data by calling the RPC method
// getAddressLookupTable (or getAccountInfo on the ALT account and parsing
// the state). Pass a slice of LoadedAddressLookupTable to NewMessageV0 or
// Builder.BuildV0 so the builder can map instruction accounts to their
// table slot indices.
type LoadedAddressLookupTable struct {
	AccountKey PublicKey   // On-chain ALT address
	Addresses  []PublicKey // All addresses in the table, ordered by slot index
}

// staticEntry tracks one static account's cumulative signer/writable
// roles, plus the order it first appeared in the instruction stream.
// firstSeen preserves the solana-web3.js-compatible bucket ordering.
type staticEntry struct {
	pubkey    PublicKey
	signer    bool
	writable  bool
	firstSeen int
}

// staticEntries is a deduplicating collector for static account keys.
// addOrUpdate is OR-semantics: the same key seen as signer in one
// instruction and non-signer in another ends up signer.
type staticEntries struct {
	m       map[PublicKey]*staticEntry
	counter int
}

func newStaticEntries() *staticEntries {
	return &staticEntries{m: make(map[PublicKey]*staticEntry)}
}

func (s *staticEntries) addOrUpdate(pk PublicKey, signer, writable bool) {
	e, ok := s.m[pk]
	if !ok {
		e = &staticEntry{pubkey: pk, firstSeen: s.counter}
		s.counter++
		s.m[pk] = e
	}
	e.signer = e.signer || signer
	e.writable = e.writable || writable
}

// bucketAndOrder returns the static account list sorted into the four
// Solana role buckets — signer-writable, signer-readonly,
// non-signer-writable, non-signer-readonly — plus the matching
// MessageHeader counters. Within each bucket entries are ordered by
// firstSeen so output is stable and matches solana-web3.js.
func (s *staticEntries) bucketAndOrder() ([]PublicKey, MessageHeader) {
	var bucketCounts [4]int
	for _, e := range s.m {
		bucketCounts[bucketOf(e)]++
	}
	var buckets [4][]*staticEntry
	for i := range buckets {
		buckets[i] = make([]*staticEntry, 0, bucketCounts[i])
	}
	for _, e := range s.m {
		b := bucketOf(e)
		buckets[b] = append(buckets[b], e)
	}
	for i := range buckets {
		sort.SliceStable(buckets[i], func(a, b int) bool {
			return buckets[i][a].firstSeen < buckets[i][b].firstSeen
		})
	}

	keys := make([]PublicKey, 0, len(s.m))
	for _, b := range buckets {
		for _, e := range b {
			keys = append(keys, e.pubkey)
		}
	}
	header := MessageHeader{
		NumRequiredSignatures:       uint8(len(buckets[0]) + len(buckets[1])),
		NumReadonlySignedAccounts:   uint8(len(buckets[1])),
		NumReadonlyUnsignedAccounts: uint8(len(buckets[3])),
	}
	return keys, header
}

// bucketOf maps a staticEntry to its role bucket index:
// 0=signer-writable, 1=signer-readonly, 2=non-signer-writable,
// 3=non-signer-readonly.
func bucketOf(e *staticEntry) int {
	switch {
	case e.signer && e.writable:
		return 0
	case e.signer && !e.writable:
		return 1
	case !e.signer && e.writable:
		return 2
	default:
		return 3
	}
}

// compileInstructions resolves each Instruction's program id and
// AccountMetas to indices into keyIndex, producing the compiled form
// stored in the message. fnName names the calling builder for error
// messages.
func compileInstructions(instructions []Instruction, keyIndex map[PublicKey]int, fnName string) ([]CompiledInstruction, error) {
	compiled := make([]CompiledInstruction, 0, len(instructions))
	for _, ix := range instructions {
		programIdx, ok := keyIndex[ix.ProgramID()]
		if !ok {
			return nil, fmt.Errorf("solana: %s: program id %s missing from key index", fnName, ix.ProgramID())
		}
		metas := ix.Accounts()
		accIdxs := make(Uint8Slice, len(metas))
		for i, m := range metas {
			idx, ok := keyIndex[m.PublicKey]
			if !ok {
				return nil, fmt.Errorf("solana: %s: account %s missing from key index", fnName, m.PublicKey)
			}
			accIdxs[i] = uint8(idx)
		}
		data, err := ix.Data()
		if err != nil {
			return nil, fmt.Errorf("solana: %s: instruction data: %w", fnName, err)
		}
		compiled = append(compiled, CompiledInstruction{
			ProgramIDIndex: uint8(programIdx),
			Accounts:       accIdxs,
			Data:           data,
		})
	}
	return compiled, nil
}

// NewMessage compiles a slice of typed Instructions into a legacy
// Message ready to be signed by Transaction.Sign.
//
// It deduplicates account keys across all instructions, orders them
// by role according to the Solana protocol, computes the header
// counters, and resolves each instruction's account list to
// positional indices into the deduplicated key array.
//
// Account ordering within the resulting Message follows the Solana
// convention:
//
//  1. payer (always first, signer + writable)
//  2. other signer-writable accounts
//  3. signer-readonly accounts
//  4. non-signer-writable accounts
//  5. non-signer-readonly accounts (including program ids)
//
// Within each bucket, accounts are ordered by the order they first
// appear in the instructions slice, which matches solana-web3.js
// and produces stable, review-friendly output.
//
// The payer is always placed first in the AccountKeys slice and is
// guaranteed to be signer + writable regardless of how individual
// instructions reference it. Roles elsewhere are cumulative: if
// the same account appears as signer in one instruction and
// non-signer in another, it ends up in the signer bucket.
func NewMessage(payer PublicKey, instructions []Instruction, recentBlockhash Hash) (*Message, error) {
	if len(instructions) == 0 {
		return nil, fmt.Errorf("solana: NewMessage: no instructions")
	}

	entries := newStaticEntries()
	entries.addOrUpdate(payer, true, true) // payer always signer+writable

	for _, ix := range instructions {
		if ix == nil {
			return nil, fmt.Errorf("solana: NewMessage: nil instruction")
		}
		for _, meta := range ix.Accounts() {
			if meta == nil {
				return nil, fmt.Errorf("solana: NewMessage: nil AccountMeta")
			}
			entries.addOrUpdate(meta.PublicKey, meta.IsSigner, meta.IsWritable)
		}
		entries.addOrUpdate(ix.ProgramID(), false, false)
	}

	accountKeys, header := entries.bucketAndOrder()
	if len(accountKeys) > 256 {
		return nil, fmt.Errorf("solana: NewMessage: %d account keys exceeds Solana maximum of 256", len(accountKeys))
	}

	keyIndex := make(map[PublicKey]int, len(accountKeys))
	for i, k := range accountKeys {
		keyIndex[k] = i
	}

	compiled, err := compileInstructions(instructions, keyIndex, "NewMessage")
	if err != nil {
		return nil, err
	}

	return &Message{
		Version:         MessageVersionLegacy,
		Header:          header,
		AccountKeys:     accountKeys,
		RecentBlockhash: recentBlockhash,
		Instructions:    compiled,
	}, nil
}

// NewMessageV0 compiles a v0 versioned Message that uses Address Lookup
// Tables to extend the account limit beyond the legacy 35-account cap.
//
// tables contains resolved ALTs: each entry holds the table's on-chain
// address and the full ordered list of addresses stored in it. The builder
// inspects every instruction account and, when the account appears in a
// table AND is not required to sign (signer accounts must always be static),
// routes it through that table rather than the static account list.
//
// Account index layout in the compiled message:
//
//	[0 .. staticLen-1]             — static accounts (same role ordering as NewMessage)
//	[staticLen .. +Σwritable]      — writable table accounts, table by table
//	[above .. +Σreadonly]          — readonly table accounts, table by table
//
// If an account appears in multiple tables the first matching table wins.
func NewMessageV0(payer PublicKey, instructions []Instruction, recentBlockhash Hash, tables []LoadedAddressLookupTable) (*Message, error) {
	if len(instructions) == 0 {
		return nil, fmt.Errorf("solana: NewMessageV0: no instructions")
	}

	// Reverse map pubkey → (tableIdx, slotIdx) from all tables.
	type tableSlot struct {
		tableIdx int
		slotIdx  int
	}
	tableIndex := make(map[PublicKey]tableSlot)
	for ti, t := range tables {
		for si, addr := range t.Addresses {
			if _, exists := tableIndex[addr]; !exists {
				tableIndex[addr] = tableSlot{tableIdx: ti, slotIdx: si}
			}
		}
	}

	// Signers must always be static, even if present in a table.
	signerSet := make(map[PublicKey]bool)
	signerSet[payer] = true
	for _, ix := range instructions {
		if ix == nil {
			return nil, fmt.Errorf("solana: NewMessageV0: nil instruction")
		}
		for _, meta := range ix.Accounts() {
			if meta == nil {
				return nil, fmt.Errorf("solana: NewMessageV0: nil AccountMeta")
			}
			if meta.IsSigner {
				signerSet[meta.PublicKey] = true
			}
		}
	}

	entries := newStaticEntries()
	entries.addOrUpdate(payer, true, true)

	tableAccounts := make([]map[int]bool, len(tables)) // tableIdx → slotIdx → writable
	for i := range tableAccounts {
		tableAccounts[i] = make(map[int]bool)
	}
	addOrUpdateTable := func(pk PublicKey, writable bool) {
		slot := tableIndex[pk]
		prev := tableAccounts[slot.tableIdx][slot.slotIdx]
		tableAccounts[slot.tableIdx][slot.slotIdx] = prev || writable
	}

	for _, ix := range instructions {
		for _, meta := range ix.Accounts() {
			_, inTable := tableIndex[meta.PublicKey]
			if inTable && !signerSet[meta.PublicKey] {
				addOrUpdateTable(meta.PublicKey, meta.IsWritable)
			} else {
				entries.addOrUpdate(meta.PublicKey, meta.IsSigner, meta.IsWritable)
			}
		}
		// Program IDs are always static.
		entries.addOrUpdate(ix.ProgramID(), false, false)
	}

	staticKeys, header := entries.bucketAndOrder()
	if len(staticKeys) > 256 {
		return nil, fmt.Errorf("solana: NewMessageV0: %d static account keys exceeds maximum of 256", len(staticKeys))
	}

	// Build full index map matching the Solana resolution order: static
	// keys, then ALL writable table accounts across every table, then ALL
	// readonly table accounts across every table. This must match
	// Message.ResolvedAccountKeys; per-table interleaving would silently
	// corrupt compiled instruction account indices.
	keyIndex := make(map[PublicKey]int, len(staticKeys))
	for i, k := range staticKeys {
		keyIndex[k] = i
	}

	type tableSlots struct {
		writable, readonly []int
	}
	perTable := make([]tableSlots, len(tables))
	emit := make([]bool, len(tables))
	for ti := range tables {
		slots := tableAccounts[ti]
		if len(slots) == 0 {
			continue
		}
		emit[ti] = true
		ts := tableSlots{}
		for slotIdx, writable := range slots {
			if writable {
				ts.writable = append(ts.writable, slotIdx)
			} else {
				ts.readonly = append(ts.readonly, slotIdx)
			}
		}
		sort.Ints(ts.writable)
		sort.Ints(ts.readonly)
		perTable[ti] = ts
	}

	next := len(staticKeys)
	for ti, t := range tables {
		if !emit[ti] {
			continue
		}
		for _, s := range perTable[ti].writable {
			keyIndex[t.Addresses[s]] = next
			next++
		}
	}
	for ti, t := range tables {
		if !emit[ti] {
			continue
		}
		for _, s := range perTable[ti].readonly {
			keyIndex[t.Addresses[s]] = next
			next++
		}
	}

	var lookups []MessageAddressTableLookup
	for ti, t := range tables {
		if !emit[ti] {
			continue
		}
		writableIdxs := make([]uint8, len(perTable[ti].writable))
		for i, s := range perTable[ti].writable {
			writableIdxs[i] = uint8(s)
		}
		readonlyIdxs := make([]uint8, len(perTable[ti].readonly))
		for i, s := range perTable[ti].readonly {
			readonlyIdxs[i] = uint8(s)
		}
		lookups = append(lookups, MessageAddressTableLookup{
			AccountKey:      t.AccountKey,
			WritableIndexes: writableIdxs,
			ReadonlyIndexes: readonlyIdxs,
		})
	}

	compiled, err := compileInstructions(instructions, keyIndex, "NewMessageV0")
	if err != nil {
		return nil, err
	}

	return &Message{
		Version:             MessageVersion0,
		Header:              header,
		AccountKeys:         staticKeys,
		RecentBlockhash:     recentBlockhash,
		Instructions:        compiled,
		AddressTableLookups: lookups,
	}, nil
}
