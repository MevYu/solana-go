// Package addresslookuptable provides typed instruction builders
// for the Solana Address Lookup Table program. Address lookup
// tables are the mechanism v0 transactions use to reference
// accounts without consuming signature space: a single 32-byte
// lookup table address in the transaction header resolves to up
// to 256 accounts on-chain, letting programs like Jupiter and
// Whirlpool route through many more accounts than a legacy
// transaction would allow.
//
// A typical lifecycle is:
//
//  1. NewCreateLookupTable(authority, payer, recentSlot)
//  2. NewExtendLookupTable(table, authority, payer, [addresses...])
//     (repeated, as many times as needed to fill the table)
//  3. transactions reference the table via MessageAddressTableLookup
//  4. NewFreezeLookupTable when the table is complete, to prevent
//     further extends
//  5. eventually NewDeactivateLookupTable + NewCloseLookupTable to
//     reclaim rent
package addresslookuptable

import "github.com/MevYu/solana-go"

// ProgramID is the canonical address of the Solana Address Lookup
// Table program.
var ProgramID = solana.MustPublicKey("AddressLookupTab1e1111111111111111111111111")
