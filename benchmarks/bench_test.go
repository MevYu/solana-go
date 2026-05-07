package benchmarks

import (
	"testing"

	"github.com/MevYu/solana-go/jsonrpc"
)

// Target structs are intentionally minimal — they capture the field
// shapes a typed client wrapper would expose, so decode cost is
// measured against realistic Go receivers rather than map[string]any.

type accountInfoResult struct {
	Context struct {
		APIVersion string `json:"apiVersion"`
		Slot       uint64 `json:"slot"`
	} `json:"context"`
	Value struct {
		Data       []string `json:"data"`
		Executable bool     `json:"executable"`
		Lamports   uint64   `json:"lamports"`
		Owner      string   `json:"owner"`
		RentEpoch  uint64   `json:"rentEpoch"`
		Space      uint64   `json:"space"`
	} `json:"value"`
}

type txMessage struct {
	AccountKeys []string `json:"accountKeys"`
	Header      struct {
		NumReadonlySignedAccounts   uint8 `json:"numReadonlySignedAccounts"`
		NumReadonlyUnsignedAccounts uint8 `json:"numReadonlyUnsignedAccounts"`
		NumRequiredSignatures       uint8 `json:"numRequiredSignatures"`
	} `json:"header"`
	Instructions []struct {
		Accounts       []int  `json:"accounts"`
		Data           string `json:"data"`
		ProgramIDIndex int    `json:"programIdIndex"`
		StackHeight    *int   `json:"stackHeight"`
	} `json:"instructions"`
	RecentBlockhash string `json:"recentBlockhash"`
}

type txMeta struct {
	ComputeUnitsConsumed uint64   `json:"computeUnitsConsumed"`
	Err                  any      `json:"err"`
	Fee                  uint64   `json:"fee"`
	LogMessages          []string `json:"logMessages"`
	PostBalances         []uint64 `json:"postBalances"`
	PreBalances          []uint64 `json:"preBalances"`
}

type encodedTransaction struct {
	Message    txMessage `json:"message"`
	Signatures []string  `json:"signatures"`
}

type transactionResult struct {
	BlockTime   int64              `json:"blockTime"`
	Meta        txMeta             `json:"meta"`
	Slot        uint64             `json:"slot"`
	Transaction encodedTransaction `json:"transaction"`
	Version     any                `json:"version"`
}

type blockResult struct {
	BlockHeight       uint64 `json:"blockHeight"`
	BlockTime         int64  `json:"blockTime"`
	Blockhash         string `json:"blockhash"`
	ParentSlot        uint64 `json:"parentSlot"`
	PreviousBlockhash string `json:"previousBlockhash"`
	Transactions      []struct {
		Meta        txMeta             `json:"meta"`
		Transaction encodedTransaction `json:"transaction"`
		Version     any                `json:"version"`
	} `json:"transactions"`
}

func BenchmarkGetAccountInfo_Decode(b *testing.B) {
	codec := jsonrpc.GoJSONCodec()
	data := GetAccountInfoJSON()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out accountInfoResult
		if err := codec.Unmarshal(data, &out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAccountInfo_Decode_StdCodec(b *testing.B) {
	codec := jsonrpc.StdCodec()
	data := GetAccountInfoJSON()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out accountInfoResult
		if err := codec.Unmarshal(data, &out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetTransaction_Decode(b *testing.B) {
	codec := jsonrpc.GoJSONCodec()
	data := GetTransactionJSON()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out transactionResult
		if err := codec.Unmarshal(data, &out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetTransaction_Decode_StdCodec(b *testing.B) {
	codec := jsonrpc.StdCodec()
	data := GetTransactionJSON()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out transactionResult
		if err := codec.Unmarshal(data, &out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetBlock_Decode(b *testing.B) {
	codec := jsonrpc.GoJSONCodec()
	data := GetBlockJSON()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out blockResult
		if err := codec.Unmarshal(data, &out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetBlock_Decode_StdCodec(b *testing.B) {
	codec := jsonrpc.StdCodec()
	data := GetBlockJSON()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out blockResult
		if err := codec.Unmarshal(data, &out); err != nil {
			b.Fatal(err)
		}
	}
}

// ---------------------------------------------------------------------
// Smoke tests: every fixture must decode with both codecs without
// error, so any drift in fixture shape is caught in CI.
// ---------------------------------------------------------------------

func TestFixtures_DecodeWithBothCodecs(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		into func() any
	}{
		{"getAccountInfo", GetAccountInfoJSON(), func() any { return &accountInfoResult{} }},
		{"getTransaction", GetTransactionJSON(), func() any { return &transactionResult{} }},
		{"getBlock", GetBlockJSON(), func() any { return &blockResult{} }},
	}
	codecs := []struct {
		name  string
		codec jsonrpc.Codec
	}{
		{"GoJSON", jsonrpc.GoJSONCodec()},
		{"Std", jsonrpc.StdCodec()},
	}
	for _, tc := range cases {
		for _, c := range codecs {
			t.Run(tc.name+"/"+c.name, func(t *testing.T) {
				out := tc.into()
				if err := c.codec.Unmarshal(tc.data, out); err != nil {
					t.Fatalf("%s via %s: %v", tc.name, c.name, err)
				}
			})
		}
	}
}
