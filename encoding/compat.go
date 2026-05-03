package encoding

// This file provides compatibility shims for code migrating from the
// github.com/gagliardetto/binary decoder. They are thin aliases over the
// canonical API in bincode.go and add no new behaviour.

// NewBinDecoder returns a Decoder reading from data. It is an alias for
// NewDecoder and exists so the gagliardetto/binary idiom
//
//	d := encoding.NewBinDecoder(data)
//	d.DecodeTo(&v)
//
// compiles unchanged against this package.
func NewBinDecoder(data []byte) *Decoder { return NewDecoder(data) }

// Position returns the number of bytes consumed so far. It is an alias for
// Pos, using the gagliardetto/binary spelling.
func (d *Decoder) Position() int { return d.pos }
