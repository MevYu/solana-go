package solana

import (
	"encoding/json"
	"testing"
)

func TestMessageVersion_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want MessageVersion
	}{
		{"legacy_string", `"legacy"`, MessageVersionLegacy},
		{"empty_string", `""`, MessageVersionLegacy},
		{"null", `null`, MessageVersionLegacy},
		{"v0", `0`, MessageVersion0},
		{"v1_unsupported_but_decodable", `1`, MessageVersion(1)},
		{"v255_clamp_max", `255`, MessageVersion(255)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var v MessageVersion
			if err := json.Unmarshal([]byte(tc.in), &v); err != nil {
				t.Fatalf("Unmarshal(%q): %v", tc.in, err)
			}
			if v != tc.want {
				t.Errorf("got %d, want %d", v, tc.want)
			}
		})
	}
}

func TestMessageVersion_UnmarshalJSON_Invalid(t *testing.T) {
	cases := []string{
		`"unknown"`, // unknown string
		`-1`,        // out of uint8 range
		`256`,       // out of uint8 range
		`1.5`,       // not an integer
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			var v MessageVersion
			if err := json.Unmarshal([]byte(in), &v); err == nil {
				t.Errorf("expected error for %q, got %d", in, v)
			}
		})
	}
}

func TestMessageVersion_MarshalJSON(t *testing.T) {
	cases := []struct {
		in   MessageVersion
		want string
	}{
		{MessageVersionLegacy, `"legacy"`},
		{MessageVersion0, `0`},
		{MessageVersion(7), `7`},
	}
	for _, tc := range cases {
		got, err := json.Marshal(tc.in)
		if err != nil {
			t.Fatalf("Marshal(%d): %v", tc.in, err)
		}
		if string(got) != tc.want {
			t.Errorf("Marshal(%d) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestMessageVersion_RoundTrip(t *testing.T) {
	for _, v := range []MessageVersion{MessageVersionLegacy, MessageVersion0, MessageVersion(42)} {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var back MessageVersion
		if err := json.Unmarshal(raw, &back); err != nil {
			t.Fatal(err)
		}
		if back != v {
			t.Errorf("round-trip %d -> %s -> %d", v, raw, back)
		}
	}
}
