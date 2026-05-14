package helpers

import "crypto/sha256"

// AnchorSighash returns the first 8 bytes of sha256(namespace + ":" + name).
// Anchor uses this scheme for several discriminators; AnchorMethodDisc and
// AnchorAccountDisc wrap it with the common namespaces. Exposed for
// non-standard namespaces (e.g. "state:" on legacy state methods).
func AnchorSighash(namespace, name string) [8]byte {
	h := sha256.New()
	h.Write([]byte(namespace))
	h.Write([]byte{':'})
	h.Write([]byte(name))
	var out [8]byte
	copy(out[:], h.Sum(nil)[:8])
	return out
}

// AnchorMethodDisc returns the 8-byte discriminator Anchor emits at the
// start of an instruction's data: sha256("global:" + name)[:8]. name is
// the snake_case Rust method on the program (e.g. "initialize", "swap").
func AnchorMethodDisc(name string) [8]byte {
	return AnchorSighash("global", name)
}

// AnchorAccountDisc returns the 8-byte tag Anchor writes at the start of
// a program account's data: sha256("account:" + name)[:8]. name is the
// PascalCase Rust struct (e.g. "Pool", "UserPosition").
func AnchorAccountDisc(name string) [8]byte {
	return AnchorSighash("account", name)
}
