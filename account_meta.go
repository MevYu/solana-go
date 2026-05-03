package solana

// AccountsSettable is implemented by types that accept a full
// replacement of their account meta list.
type AccountsSettable interface {
	SetAccounts(accounts []*AccountMeta) error
}

// AccountsGettable is implemented by types that can return their
// current list of account metas.
type AccountsGettable interface {
	GetAccounts() []*AccountMeta
}

// Meta creates a new AccountMeta for pubKey with IsSigner and
// IsWritable both false. Chain WRITE() and SIGNER() to promote
// the account:
//
//	solana.Meta(pk).WRITE().SIGNER()
func Meta(pubKey PublicKey) *AccountMeta {
	return &AccountMeta{PublicKey: pubKey}
}

// WRITE sets IsWritable to true and returns the receiver for
// chaining.
func (m *AccountMeta) WRITE() *AccountMeta {
	m.IsWritable = true
	return m
}

// SIGNER sets IsSigner to true and returns the receiver for
// chaining.
func (m *AccountMeta) SIGNER() *AccountMeta {
	m.IsSigner = true
	return m
}

// Less reports whether m should sort before other in account-meta
// lists: signers before non-signers, then writable before
// read-only.
func (m *AccountMeta) Less(other *AccountMeta) bool {
	if m.IsSigner != other.IsSigner {
		return m.IsSigner
	}
	if m.IsWritable != other.IsWritable {
		return m.IsWritable
	}
	return false
}

// ---------------------------------------------------------------
// AccountMetaSlice
// ---------------------------------------------------------------

// AccountMetaSlice is a convenience wrapper around a slice of
// AccountMeta pointers. It provides helpers for common operations
// like appending, filtering, and extracting keys.
type AccountMetaSlice []*AccountMeta

// Append adds an account to the slice.
func (s *AccountMetaSlice) Append(account *AccountMeta) {
	*s = append(*s, account)
}

// SetAccounts replaces the slice contents with accounts.
func (s *AccountMetaSlice) SetAccounts(accounts []*AccountMeta) error {
	*s = accounts
	return nil
}

// GetAccounts returns all non-nil entries in the slice.
func (s AccountMetaSlice) GetAccounts() []*AccountMeta {
	out := make([]*AccountMeta, 0, len(s))
	for _, m := range s {
		if m != nil {
			out = append(out, m)
		}
	}
	return out
}

// Get returns the AccountMeta at the given index, or nil if the
// index is out of range.
func (s AccountMetaSlice) Get(index int) *AccountMeta {
	if index >= 0 && index < len(s) {
		return s[index]
	}
	return nil
}

// GetSigners returns only the entries that have IsSigner set.
func (s AccountMetaSlice) GetSigners() []*AccountMeta {
	out := make([]*AccountMeta, 0, len(s))
	for _, m := range s {
		if m != nil && m.IsSigner {
			out = append(out, m)
		}
	}
	return out
}

// GetKeys returns the public keys of all entries.
func (s AccountMetaSlice) GetKeys() []PublicKey {
	keys := make([]PublicKey, 0, len(s))
	for _, m := range s {
		if m != nil {
			keys = append(keys, m.PublicKey)
		}
	}
	return keys
}

// Len returns the length of the slice.
func (s AccountMetaSlice) Len() int {
	return len(s)
}
