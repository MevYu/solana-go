package solana

import (
	"encoding/json"
	"testing"

	"github.com/mr-tron/base58"
)

// getTransaction/getBlock return meta.innerInstructions[].instructions[].data
// as base58. Before CompiledInstruction.Data was typed Base58Data, the
// encoding/json default decoded []byte as base64 and failed with
// "illegal base64 data ..." on every transaction containing CPIs.
func TestCompiledInstructionUnmarshalJSONBase58Data(t *testing.T) {
	payload := []byte{3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 42}
	dataB58 := base58.Encode(payload)

	raw := `{"programIdIndex":4,"accounts":[1,2,3],"data":"` + dataB58 + `","stackHeight":2}`

	var ci CompiledInstruction
	if err := json.Unmarshal([]byte(raw), &ci); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if ci.ProgramIDIndex != 4 {
		t.Errorf("ProgramIDIndex = %d, want 4", ci.ProgramIDIndex)
	}
	if got := []uint8(ci.Accounts); len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Errorf("Accounts = %v, want [1 2 3]", got)
	}
	if string(ci.Data) != string(payload) {
		t.Errorf("Data = %x, want %x", ci.Data, payload)
	}

	// round-trip: data must re-marshal as base58, not base64
	out, err := json.Marshal(ci)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var back CompiledInstruction
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("re-unmarshal failed: %v", err)
	}
	if string(back.Data) != string(payload) {
		t.Errorf("round-trip Data = %x, want %x", back.Data, payload)
	}
}
