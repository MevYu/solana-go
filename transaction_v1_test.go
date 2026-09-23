package solana

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
)

func TestTransactionV1_RecordedSignature(t *testing.T) {
	wire, err := os.ReadFile("testdata/v1_transaction.bin")
	if err != nil {
		t.Fatal(err)
	}
	var tx Transaction
	if err := tx.UnmarshalBinary(wire); err != nil {
		t.Fatalf("decode v1 transaction: %v", err)
	}
	if tx.Message.Version != MessageVersion1 || len(tx.Signatures) != 1 {
		t.Fatalf("version = %d, signatures = %d", tx.Message.Version, len(tx.Signatures))
	}
	const signature = "2psiRktzwwp24HJBUySfduJchgeKoHYMEoEK2FvzX4DMYyXxEtRqhGXmPH8YWDcY6a1J6LZDbbBfzrXf9ZYSCf9v"
	if got := tx.Signatures[0].String(); got != signature {
		t.Fatalf("signature = %s", got)
	}
	if len(tx.Message.AccountKeys) != 37 || len(tx.Message.Instructions) != 4 {
		t.Fatalf("accounts = %d, instructions = %d", len(tx.Message.AccountKeys), len(tx.Message.Instructions))
	}
	c := tx.Message.TransactionConfig
	if c == nil || c.PriorityFee == nil || *c.PriorityFee != 0 ||
		c.ComputeUnitLimit == nil || *c.ComputeUnitLimit != 522660 ||
		c.LoadedAccountsDataSizeLimit == nil || *c.LoadedAccountsDataSizeLimit != 7766016 || c.HeapSize != nil {
		t.Fatalf("config = %+v", c)
	}
	if err := tx.VerifySignatures(); err != nil {
		t.Fatalf("verify signature: %v", err)
	}
	got, err := tx.Marshal()
	if err != nil {
		t.Fatalf("re-encode v1 transaction: %v", err)
	}
	if !bytes.Equal(got, wire) {
		t.Fatal("v1 transaction changed on re-encode")
	}
	encoded, err := json.Marshal([2]string{base64.StdEncoding.EncodeToString(wire), "base64"})
	if err != nil {
		t.Fatal(err)
	}
	var fromJSON Transaction
	if err := json.Unmarshal(encoded, &fromJSON); err != nil {
		t.Fatalf("decode RPC tuple: %v", err)
	}
	if fromJSON.Signatures[0] != tx.Signatures[0] {
		t.Fatal("RPC tuple signature mismatch")
	}
}

func TestTransactionV1_RejectsTruncatedSignature(t *testing.T) {
	wire, err := os.ReadFile("testdata/v1_transaction.bin")
	if err != nil {
		t.Fatal(err)
	}
	if err := new(Transaction).UnmarshalBinary(wire[:len(wire)-1]); err == nil {
		t.Fatal("expected truncated v1 signature to fail")
	}
}
