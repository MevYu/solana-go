package jsonrpc

import (
	"encoding/json"
	"testing"
)

// benchTx is a compact stand-in for a Solana transaction summary as
// it appears inside a getBlock response. It is not the real shape,
// but it is a realistic mix of numbers, strings, and nested arrays
// that exercises JSON decoding on the same kinds of fields a real
// getBlock contains.
type benchTx struct {
	Signatures []string `json:"signatures"`
	Slot       uint64   `json:"slot"`
	Fee        uint64   `json:"fee"`
	Status     string   `json:"status"`
	Logs       []string `json:"logs"`
	Accounts   []int    `json:"accounts"`
}

type benchBlock struct {
	Blockhash    string    `json:"blockhash"`
	ParentSlot   uint64    `json:"parentSlot"`
	Transactions []benchTx `json:"transactions"`
}

// makeBenchJSON synthesises a ~100 transaction block and marshals it
// once via stdlib encoding/json. The result is reused as the input
// to every codec benchmark so the comparison is apples-to-apples.
func makeBenchJSON() []byte {
	b := benchBlock{
		Blockhash:  "FakeBlockHashDoNotUseItIsAPlaceholder1234567",
		ParentSlot: 123456789,
	}
	for i := 0; i < 100; i++ {
		b.Transactions = append(b.Transactions, benchTx{
			Signatures: []string{
				"5Nqmx3cCMvsQGFnRoHxJyvsrtTBUEYqKtZjKz6pFakeSigOne",
				"5Nqmx3cCMvsQGFnRoHxJyvsrtTBUEYqKtZjKz6pFakeSigTwo",
			},
			Slot:   uint64(100_000_000 + i),
			Fee:    5000,
			Status: "Success",
			Logs: []string{
				"Program 11111111111111111111111111111111 invoke [1]",
				"Program 11111111111111111111111111111111 success",
			},
			Accounts: []int{0, 1, 2, 3, 4},
		})
	}
	data, _ := json.Marshal(b)
	return data
}

var benchJSONData = makeBenchJSON()

// These benchmarks establish the baseline for the JSON-library
// decision (goccy/go-json vs stdlib). Both codecs decode the same
// input bytes into the same target type, so the only variable is the
// library itself.

func BenchmarkCodec_Decode_StdCodec(b *testing.B) {
	codec := StdCodec()
	b.SetBytes(int64(len(benchJSONData)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out benchBlock
		if err := codec.Unmarshal(benchJSONData, &out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodec_Decode_GoJSONCodec(b *testing.B) {
	codec := GoJSONCodec()
	b.SetBytes(int64(len(benchJSONData)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out benchBlock
		if err := codec.Unmarshal(benchJSONData, &out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodec_Encode_StdCodec(b *testing.B) {
	codec := StdCodec()
	var src benchBlock
	_ = json.Unmarshal(benchJSONData, &src)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := codec.Marshal(&src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodec_Encode_GoJSONCodec(b *testing.B) {
	codec := GoJSONCodec()
	var src benchBlock
	_ = json.Unmarshal(benchJSONData, &src)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := codec.Marshal(&src); err != nil {
			b.Fatal(err)
		}
	}
}
