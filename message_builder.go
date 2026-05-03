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

	type entry struct {
		pubkey    PublicKey
		signer    bool
		writable  bool
		firstSeen int
	}

	entries := make(map[PublicKey]*entry)
	firstSeenCounter := 0

	addOrUpdate := func(pk PublicKey, signer, writable bool) {
		e, ok := entries[pk]
		if !ok {
			e = &entry{pubkey: pk, firstSeen: firstSeenCounter}
			firstSeenCounter++
			entries[pk] = e
		}
		e.signer = e.signer || signer
		e.writable = e.writable || writable
	}

	// Payer is always added first with signer + writable role.
	addOrUpdate(payer, true, true)

	for _, ix := range instructions {
		if ix == nil {
			return nil, fmt.Errorf("solana: NewMessage: nil instruction")
		}
		for _, meta := range ix.Accounts() {
			if meta == nil {
				return nil, fmt.Errorf("solana: NewMessage: nil AccountMeta")
			}
			addOrUpdate(meta.PublicKey, meta.IsSigner, meta.IsWritable)
		}
		addOrUpdate(ix.ProgramID(), false, false)
	}

	// Bucket by role: 0=sw, 1=sr, 2=nw, 3=nr.
	// Two-pass: count first so each bucket slice is pre-sized exactly.
	var bucketCounts [4]int
	for _, e := range entries {
		switch {
		case e.signer && e.writable:
			bucketCounts[0]++
		case e.signer && !e.writable:
			bucketCounts[1]++
		case !e.signer && e.writable:
			bucketCounts[2]++
		default:
			bucketCounts[3]++
		}
	}
	var buckets [4][]*entry
	for i := range buckets {
		buckets[i] = make([]*entry, 0, bucketCounts[i])
	}
	for _, e := range entries {
		switch {
		case e.signer && e.writable:
			buckets[0] = append(buckets[0], e)
		case e.signer && !e.writable:
			buckets[1] = append(buckets[1], e)
		case !e.signer && e.writable:
			buckets[2] = append(buckets[2], e)
		default:
			buckets[3] = append(buckets[3], e)
		}
	}
	for i := range buckets {
		sort.SliceStable(buckets[i], func(a, b int) bool {
			return buckets[i][a].firstSeen < buckets[i][b].firstSeen
		})
	}

	accountKeys := make([]PublicKey, 0, len(entries))
	for _, b := range buckets {
		for _, e := range b {
			accountKeys = append(accountKeys, e.pubkey)
		}
	}

	if len(accountKeys) > 256 {
		return nil, fmt.Errorf("solana: NewMessage: %d account keys exceeds Solana maximum of 256", len(accountKeys))
	}

	header := MessageHeader{
		NumRequiredSignatures:       uint8(len(buckets[0]) + len(buckets[1])),
		NumReadonlySignedAccounts:   uint8(len(buckets[1])),
		NumReadonlyUnsignedAccounts: uint8(len(buckets[3])),
	}

	keyIndex := make(map[PublicKey]int, len(accountKeys))
	for i, k := range accountKeys {
		keyIndex[k] = i
	}

	compiled := make([]CompiledInstruction, 0, len(instructions))
	for _, ix := range instructions {
		programIdx, ok := keyIndex[ix.ProgramID()]
		if !ok {
			return nil, fmt.Errorf("solana: NewMessage: program id %s missing from key index", ix.ProgramID())
		}
		metas := ix.Accounts()
		accIdxs := make(Uint8Slice, len(metas))
		for i, m := range metas {
			idx, ok := keyIndex[m.PublicKey]
			if !ok {
				return nil, fmt.Errorf("solana: NewMessage: account %s missing from key index", m.PublicKey)
			}
			accIdxs[i] = uint8(idx)
		}
		data, err := ix.Data()
		if err != nil {
			return nil, fmt.Errorf("solana: NewMessage: instruction data: %w", err)
		}
		compiled = append(compiled, CompiledInstruction{
			ProgramIDIndex: uint8(programIdx),
			Accounts:       accIdxs,
			Data:           data,
		})
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

	// Build a reverse map: pubkey → (tableIndex, slotIndex) from all tables.
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

	// Collect signer pubkeys from all instructions — signers must be static.
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

	// Decide static vs table for each account.
	type entry struct {
		pubkey    PublicKey
		signer    bool
		writable  bool
		firstSeen int
	}
	staticEntries := make(map[PublicKey]*entry)
	firstSeenCounter := 0

	addOrUpdateStatic := func(pk PublicKey, signer, writable bool) {
		e, ok := staticEntries[pk]
		if !ok {
			e = &entry{pubkey: pk, firstSeen: firstSeenCounter}
			firstSeenCounter++
			staticEntries[pk] = e
		}
		e.signer = e.signer || signer
		e.writable = e.writable || writable
	}

	// tableAccounts[tableIdx] maps slotIdx → writable flag.
	type tableAccountEntry struct {
		slotIdx  int
		writable bool
	}
	tableAccounts := make([]map[int]bool, len(tables)) // tableIdx → slotIdx → writable
	for i := range tableAccounts {
		tableAccounts[i] = make(map[int]bool)
	}

	addOrUpdateTable := func(pk PublicKey, writable bool) {
		slot, ok := tableIndex[pk]
		if !ok {
			return
		}
		prev := tableAccounts[slot.tableIdx][slot.slotIdx]
		tableAccounts[slot.tableIdx][slot.slotIdx] = prev || writable
	}

	// Payer always static.
	addOrUpdateStatic(payer, true, true)

	for _, ix := range instructions {
		for _, meta := range ix.Accounts() {
			_, inTable := tableIndex[meta.PublicKey]
			if inTable && !signerSet[meta.PublicKey] {
				addOrUpdateTable(meta.PublicKey, meta.IsWritable)
			} else {
				addOrUpdateStatic(meta.PublicKey, meta.IsSigner, meta.IsWritable)
			}
		}
		// Program IDs are always static.
		addOrUpdateStatic(ix.ProgramID(), false, false)
	}

	// Sort static accounts into 4 buckets (same as NewMessage).
	// Two-pass: count first to pre-size each bucket slice exactly.
	var bucketCounts [4]int
	for _, e := range staticEntries {
		switch {
		case e.signer && e.writable:
			bucketCounts[0]++
		case e.signer && !e.writable:
			bucketCounts[1]++
		case !e.signer && e.writable:
			bucketCounts[2]++
		default:
			bucketCounts[3]++
		}
	}
	var buckets [4][]*entry
	for i := range buckets {
		buckets[i] = make([]*entry, 0, bucketCounts[i])
	}
	for _, e := range staticEntries {
		switch {
		case e.signer && e.writable:
			buckets[0] = append(buckets[0], e)
		case e.signer && !e.writable:
			buckets[1] = append(buckets[1], e)
		case !e.signer && e.writable:
			buckets[2] = append(buckets[2], e)
		default:
			buckets[3] = append(buckets[3], e)
		}
	}
	for i := range buckets {
		sort.SliceStable(buckets[i], func(a, b int) bool {
			return buckets[i][a].firstSeen < buckets[i][b].firstSeen
		})
	}

	staticKeys := make([]PublicKey, 0, len(staticEntries))
	for _, b := range buckets {
		for _, e := range b {
			staticKeys = append(staticKeys, e.pubkey)
		}
	}
	if len(staticKeys) > 256 {
		return nil, fmt.Errorf("solana: NewMessageV0: %d static account keys exceeds maximum of 256", len(staticKeys))
	}

	header := MessageHeader{
		NumRequiredSignatures:       uint8(len(buckets[0]) + len(buckets[1])),
		NumReadonlySignedAccounts:   uint8(len(buckets[1])),
		NumReadonlyUnsignedAccounts: uint8(len(buckets[3])),
	}

	// Build full index map: static keys first, then writable table accounts,
	// then readonly table accounts (per table, in table order).
	keyIndex := make(map[PublicKey]int, len(staticKeys))
	for i, k := range staticKeys {
		keyIndex[k] = i
	}
	next := len(staticKeys)

	var lookups []MessageAddressTableLookup
	for ti, t := range tables {
		slots := tableAccounts[ti]
		if len(slots) == 0 {
			continue
		}
		// Collect and sort writable then readonly slot indices.
		// Pre-count to avoid reallocation inside the fill loop.
		var nWritable int
		for _, writable := range slots {
			if writable {
				nWritable++
			}
		}
		writableSlots := make([]int, 0, nWritable)
		readonlySlots := make([]int, 0, len(slots)-nWritable)
		for slotIdx, writable := range slots {
			if writable {
				writableSlots = append(writableSlots, slotIdx)
			} else {
				readonlySlots = append(readonlySlots, slotIdx)
			}
		}
		sort.Ints(writableSlots)
		sort.Ints(readonlySlots)

		writableIdxs := make([]uint8, len(writableSlots))
		for i, s := range writableSlots {
			writableIdxs[i] = uint8(s)
			keyIndex[t.Addresses[s]] = next
			next++
		}
		readonlyIdxs := make([]uint8, len(readonlySlots))
		for i, s := range readonlySlots {
			readonlyIdxs[i] = uint8(s)
			keyIndex[t.Addresses[s]] = next
			next++
		}
		lookups = append(lookups, MessageAddressTableLookup{
			AccountKey:      t.AccountKey,
			WritableIndexes: writableIdxs,
			ReadonlyIndexes: readonlyIdxs,
		})
	}

	// Compile instructions.
	compiled := make([]CompiledInstruction, 0, len(instructions))
	for _, ix := range instructions {
		programIdx, ok := keyIndex[ix.ProgramID()]
		if !ok {
			return nil, fmt.Errorf("solana: NewMessageV0: program id %s missing from key index", ix.ProgramID())
		}
		metas := ix.Accounts()
		accIdxs := make(Uint8Slice, len(metas))
		for i, m := range metas {
			idx, ok := keyIndex[m.PublicKey]
			if !ok {
				return nil, fmt.Errorf("solana: NewMessageV0: account %s missing from key index", m.PublicKey)
			}
			accIdxs[i] = uint8(idx)
		}
		data, err := ix.Data()
		if err != nil {
			return nil, fmt.Errorf("solana: NewMessageV0: instruction data: %w", err)
		}
		compiled = append(compiled, CompiledInstruction{
			ProgramIDIndex: uint8(programIdx),
			Accounts:       accIdxs,
			Data:           data,
		})
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
