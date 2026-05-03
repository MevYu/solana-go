package solana

import "testing"

func TestAccount_ZeroValue(t *testing.T) {
	var a Account
	if a.Lamports != 0 {
		t.Errorf("Lamports = %d, want 0", a.Lamports)
	}
	if !a.Owner.IsZero() {
		t.Error("Owner should be zero")
	}
	if a.Data != nil {
		t.Errorf("Data = %v, want nil", a.Data)
	}
	if a.Executable {
		t.Error("Executable should be false")
	}
	if a.RentEpoch != 0 {
		t.Errorf("RentEpoch = %d, want 0", a.RentEpoch)
	}
}

func TestAccount_Fields(t *testing.T) {
	owner, err := PublicKeyFromBase58(tokenProgramID)
	if err != nil {
		t.Fatal(err)
	}
	a := Account{
		Lamports:   2_039_280,
		Owner:      owner,
		Data:       []byte{0x01, 0x02, 0x03},
		Executable: false,
		RentEpoch:  0,
	}
	if a.Lamports != 2_039_280 {
		t.Errorf("Lamports = %d", a.Lamports)
	}
	if !a.Owner.Equal(owner) {
		t.Error("Owner mismatch")
	}
	if len(a.Data) != 3 {
		t.Errorf("Data len = %d", len(a.Data))
	}
}
