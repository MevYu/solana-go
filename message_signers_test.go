package solana

import "testing"

// Signers must not panic when a message decoded from untrusted bytes
// claims more required signers than it has account keys.
func TestMessageSigners_ClampsToAccountKeys(t *testing.T) {
	m := &Message{
		AccountKeys: []PublicKey{{0x01}, {0x02}},
	}
	m.Header.NumRequiredSignatures = 5 // > len(AccountKeys)

	got := m.Signers()
	if len(got) != 2 {
		t.Fatalf("Signers len = %d, want 2 (clamped)", len(got))
	}
	if !got[0].Equal(PublicKey{0x01}) || !got[1].Equal(PublicKey{0x02}) {
		t.Errorf("Signers = %v", got)
	}
}
