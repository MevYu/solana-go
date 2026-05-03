package solana

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUint8Slice_MarshalJSON(t *testing.T) {
	cases := []struct {
		name string
		in   Uint8Slice
		want string
	}{
		{"nil", nil, "null"},
		{"empty", Uint8Slice{}, "[]"},
		{"single", Uint8Slice{42}, "[42]"},
		{"multi", Uint8Slice{0, 1, 255}, "[0,1,255]"},
		{"boundaries", Uint8Slice{0, 9, 10, 99, 100, 255}, "[0,9,10,99,100,255]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestUint8Slice_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Uint8Slice
		nil_ bool
	}{
		{"null", "null", nil, true},
		{"empty", "[]", Uint8Slice{}, false},
		{"single", "[42]", Uint8Slice{42}, false},
		{"multi", "[0,1,255]", Uint8Slice{0, 1, 255}, false},
		{"whitespace", " [ 1 , 2 , 3 ] ", Uint8Slice{1, 2, 3}, false},
		{"newlines", "[\n\t1,\n\t2\n]", Uint8Slice{1, 2}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got Uint8Slice
			if err := json.Unmarshal([]byte(tc.in), &got); err != nil {
				t.Fatal(err)
			}
			if tc.nil_ {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if len(got) != len(tc.want) {
				t.Fatalf("len mismatch: %d vs %d", len(got), len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d]: got %d, want %d", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestUint8Slice_UnmarshalJSON_Direct(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"not array number", "42"},
		{"not array string", "\"abc\""},
		{"out of range", "[256]"},
		{"negative sign", "[-1]"},
		{"float", "[1.5]"},
		{"missing close", "[1,2"},
		{"missing separator", "[1 2]"},
		{"trailing garbage", "[1]x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var s Uint8Slice
			if err := s.UnmarshalJSON([]byte(tc.in)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestUint8Slice_RoundTrip(t *testing.T) {
	orig := Uint8Slice{0, 1, 42, 127, 128, 200, 255}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatal(err)
	}
	var back Uint8Slice
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if len(back) != len(orig) {
		t.Fatalf("len mismatch: %d vs %d", len(back), len(orig))
	}
	for i := range orig {
		if back[i] != orig[i] {
			t.Errorf("[%d]: got %d, want %d", i, back[i], orig[i])
		}
	}
}

func TestCompiledInstruction_AccountsAsJSONArray(t *testing.T) {
	ix := CompiledInstruction{
		ProgramIDIndex: 7,
		Accounts:       Uint8Slice{0, 1, 2},
		Data:           []byte{0xff},
	}
	data, err := json.Marshal(ix)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\"Accounts\":[0,1,2]") {
		t.Fatalf("expected Accounts as JSON array, got %s", data)
	}
}

func TestMessageAddressTableLookup_IndexesAsJSONArrays(t *testing.T) {
	lk := MessageAddressTableLookup{
		AccountKey:      PublicKey{0x11},
		WritableIndexes: Uint8Slice{5, 6},
		ReadonlyIndexes: Uint8Slice{7},
	}
	data, err := json.Marshal(lk)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "\"WritableIndexes\":[5,6]") {
		t.Errorf("WritableIndexes should be JSON array: %s", s)
	}
	if !strings.Contains(s, "\"ReadonlyIndexes\":[7]") {
		t.Errorf("ReadonlyIndexes should be JSON array: %s", s)
	}
}
