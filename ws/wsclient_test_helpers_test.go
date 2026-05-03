package ws

import (
	"bytes"
	"crypto/ed25519"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	solana "github.com/MevYu/solana-go"

	"github.com/gorilla/websocket"
)

const (
	// systemProgramID decodes to 32 zero bytes; it is the canonical
	// base58 form of the Solana system program address.
	systemProgramID = "11111111111111111111111111111111"
	// tokenProgramID is the SPL Token program.
	tokenProgramID = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
)

// startTestWSServer boots an httptest server that upgrades incoming
// requests to WebSocket and hands them to the provided handler.
// The returned wsURL has the ws:// scheme so it can be passed
// straight to DialWebSocket.
func startTestWSServer(t *testing.T, handler func(c *websocket.Conn)) (*httptest.Server, string) {
	t.Helper()
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		handler(conn)
	}))
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	return srv, wsURL
}

func makeDeterministicKeypairWS(t testing.TB, marker byte) *solana.Ed25519Keypair {
	t.Helper()
	seed := bytes.Repeat([]byte{marker}, ed25519.SeedSize)
	kp, err := solana.Ed25519KeypairFromSeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	return kp
}

// makeSimpleTx returns an unsigned Transaction for use in ws tests.
func makeSimpleTx(t testing.TB, nSigners int) (*solana.Transaction, []*solana.Ed25519Keypair) {
	t.Helper()
	systemProgram, err := solana.PublicKeyFromBase58(systemProgramID)
	if err != nil {
		t.Fatal(err)
	}
	kps := make([]*solana.Ed25519Keypair, nSigners)
	keys := make([]solana.PublicKey, 0, nSigners+1)
	for i := 0; i < nSigners; i++ {
		kps[i] = makeDeterministicKeypairWS(t, byte(i+1))
		keys = append(keys, kps[i].PublicKey())
	}
	keys = append(keys, systemProgram)

	ixAccounts := make(solana.Uint8Slice, nSigners)
	for i := range ixAccounts {
		ixAccounts[i] = uint8(i)
	}

	msg := solana.Message{
		Version: solana.MessageVersionLegacy,
		Header: solana.MessageHeader{
			NumRequiredSignatures:       uint8(nSigners),
			NumReadonlySignedAccounts:   0,
			NumReadonlyUnsignedAccounts: 1,
		},
		AccountKeys:     keys,
		RecentBlockhash: solana.Hash{0xAA, 0xBB, 0xCC},
		Instructions: []solana.CompiledInstruction{
			{
				ProgramIDIndex: uint8(nSigners),
				Accounts:       ixAccounts,
				Data:           []byte{0x02, 0, 0, 0, 1, 2, 3, 4},
			},
		},
	}
	return solana.NewTransaction(msg), kps
}
