// Package anchor computes the 8-byte discriminators the Anchor framework
// writes at the start of instruction data and program account data. It is
// pure logic over crypto/sha256 with no dependency on the rest of the SDK.
package anchor

import "crypto/sha256"

// Sighash returns the first 8 bytes of sha256(namespace + ":" + name).
// Anchor uses this scheme for several discriminators; MethodDisc and
// AccountDisc wrap it with the common namespaces. Exposed for
// non-standard namespaces (e.g. "state:" on legacy state methods).
func Sighash(namespace, name string) [8]byte {
	h := sha256.New()
	h.Write([]byte(namespace))
	h.Write([]byte{':'})
	h.Write([]byte(name))
	var out [8]byte
	copy(out[:], h.Sum(nil)[:8])
	return out
}

// MethodDisc returns the 8-byte discriminator Anchor emits at the start of
// an instruction's data: sha256("global:" + name)[:8]. name is the
// snake_case Rust method on the program (e.g. "initialize", "swap").
func MethodDisc(name string) [8]byte {
	return Sighash("global", name)
}

// AccountDisc returns the 8-byte tag Anchor writes at the start of a
// program account's data: sha256("account:" + name)[:8]. name is the
// PascalCase Rust struct (e.g. "Pool", "UserPosition").
func AccountDisc(name string) [8]byte {
	return Sighash("account", name)
}
