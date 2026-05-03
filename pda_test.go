package solana

import (
	"testing"
)

func TestCreateProgramAddress_TooManySeeds(t *testing.T) {
	seeds := make([][]byte, MaxPDASeeds+1)
	for i := range seeds {
		seeds[i] = []byte{byte(i)}
	}
	if _, err := CreateProgramAddress(seeds, PublicKey{0x01}); err == nil {
		t.Fatal("expected error for too many seeds")
	}
}

func TestCreateProgramAddress_SeedTooLong(t *testing.T) {
	seeds := [][]byte{make([]byte, MaxPDASeedLength+1)}
	if _, err := CreateProgramAddress(seeds, PublicKey{0x01}); err == nil {
		t.Fatal("expected error for oversized seed")
	}
}

func TestFindProgramAddress_Deterministic(t *testing.T) {
	seeds := [][]byte{[]byte("test"), []byte("seed")}
	programID := PublicKey{0x42, 0x42}

	pk1, bump1, err := FindProgramAddress(seeds, programID)
	if err != nil {
		t.Fatal(err)
	}
	pk2, bump2, err := FindProgramAddress(seeds, programID)
	if err != nil {
		t.Fatal(err)
	}
	if !pk1.Equal(pk2) {
		t.Errorf("same inputs produced different PDAs: %s vs %s", pk1, pk2)
	}
	if bump1 != bump2 {
		t.Errorf("same inputs produced different bumps: %d vs %d", bump1, bump2)
	}
}

func TestFindProgramAddress_IsOffCurve(t *testing.T) {
	seeds := [][]byte{[]byte("anything")}
	programID := PublicKey{0xAA, 0xBB, 0xCC}

	pk, _, err := FindProgramAddress(seeds, programID)
	if err != nil {
		t.Fatal(err)
	}
	if isOnCurve(pk[:]) {
		t.Errorf("FindProgramAddress returned an on-curve point %s", pk)
	}
}

func TestCreateProgramAddress_RecoversFromFindBump(t *testing.T) {
	seeds := [][]byte{[]byte("hello"), []byte("world")}
	programID := PublicKey{0x01, 0x02, 0x03}

	found, bump, err := FindProgramAddress(seeds, programID)
	if err != nil {
		t.Fatal(err)
	}

	// Re-derive using the same seeds + bump via CreateProgramAddress.
	augmented := append([][]byte{}, seeds...)
	augmented = append(augmented, []byte{bump})
	redone, err := CreateProgramAddress(augmented, programID)
	if err != nil {
		t.Fatal(err)
	}
	if !found.Equal(redone) {
		t.Errorf("redo mismatch: %s vs %s", found, redone)
	}
}

func TestFindProgramAddress_DifferentSeedsDifferentPDAs(t *testing.T) {
	programID := PublicKey{0x01}
	pk1, _, err := FindProgramAddress([][]byte{[]byte("a")}, programID)
	if err != nil {
		t.Fatal(err)
	}
	pk2, _, err := FindProgramAddress([][]byte{[]byte("b")}, programID)
	if err != nil {
		t.Fatal(err)
	}
	if pk1.Equal(pk2) {
		t.Errorf("distinct seeds produced equal PDAs: %s", pk1)
	}
}

func TestIsOnCurve_RealKeys(t *testing.T) {
	// A freshly generated Ed25519 keypair's public key must be
	// on the curve by definition. Run 10 samples to make sure the
	// predicate is stable.
	for i := 0; i < 10; i++ {
		kp, err := NewEd25519Keypair()
		if err != nil {
			t.Fatal(err)
		}
		pk := kp.PublicKey()
		if !isOnCurve(pk[:]) {
			t.Errorf("real Ed25519 key reported off-curve: %s", pk)
		}
	}
}

func TestIsOnCurve_WrongLength(t *testing.T) {
	if isOnCurve([]byte{0x01}) {
		t.Error("short slice should not be on-curve")
	}
}

func TestFindProgramAddress_EmptySeeds(t *testing.T) {
	// Solana allows empty seeds (only the bump contributes).
	pk, _, err := FindProgramAddress(nil, PublicKey{0x01})
	if err != nil {
		t.Fatal(err)
	}
	if isOnCurve(pk[:]) {
		t.Errorf("empty-seed PDA should be off-curve, got %s", pk)
	}
}
