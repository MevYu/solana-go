# Getting Started

This page walks you through installing the SDK, parsing a
public key, generating a local keypair, and building / signing /
sending a transfer transaction against a devnet endpoint.

## Requirements

- **Go 1.24 or later.** The module's `go.mod` declares `go 1.24`.
- A Solana RPC endpoint. For local experiments, use
  `https://api.devnet.solana.com`.

## Install

```bash
go get github.com/MevYu/solana-go
```

Runtime dependency footprint:

- `github.com/mr-tron/base58` — base58 encoding
- `github.com/goccy/go-json` — fast JSON codec (default)
- `github.com/gorilla/websocket` — WebSocket subscriptions
- `github.com/tyler-smith/go-bip39` — BIP39 mnemonic seeds
- `filippo.io/edwards25519` — PDA off-curve check
- `github.com/klauspost/compress` (transitive) — `base64+zstd`
  account-data decoding

## Parse a public key

```go
package main

import (
    "fmt"
    "log"

    "github.com/MevYu/solana-go"
)

func main() {
    pk, err := solana.PublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(pk) // TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA
}
```

`PublicKey` is a 32-byte fixed-size array. Constructing one from a
base58 string is strict: the wrong length or a character outside
the base58 alphabet returns a wrapped `ErrInvalidLength` or
`ErrInvalidBase58`. See [Keys, Hashes, Signatures](Keys-Hashes-Signatures).

## Generate a local keypair

```go
kp, err := solana.NewEd25519Keypair()
if err != nil {
    log.Fatal(err)
}
fmt.Println("public key:", kp.PublicKey())
```

An `Ed25519Keypair` implements the [Signer](Signers) interface.
Pass it directly to `Transaction.Sign`.

## Build, sign, and send a transfer

```go
package main

import (
    "context"
    "log"

    "github.com/MevYu/solana-go"
    "github.com/MevYu/solana-go/jsonrpc"
    "github.com/MevYu/solana-go/programs/system"
    "github.com/MevYu/solana-go/rpc"
)

func main() {
    ctx := context.Background()
    c := rpc.NewClient("https://api.devnet.solana.com", jsonrpc.Config{})

    payer, _ := solana.NewEd25519Keypair()
    var recipient solana.PublicKey // fill with a real key

    // 1. Fetch the latest blockhash.
    bh, err := c.GetLatestBlockhash(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // 2. Compile the transfer instruction into a Message.
    msg, err := solana.NewMessage(
        payer.PublicKey(),
        []solana.Instruction{
            system.NewTransfer(payer.PublicKey(), recipient, 1_000_000),
        },
        bh.Blockhash,
    )
    if err != nil {
        log.Fatal(err)
    }

    // 3. Sign with the payer.
    tx := solana.NewTransaction(*msg)
    if err := tx.Sign(ctx, payer); err != nil {
        log.Fatal(err)
    }

    // 4. Broadcast.
    sig, err := c.SendTransaction(ctx, tx)
    if err != nil {
        log.Fatal(err)
    }
    log.Println("submitted:", sig)
}
```

The four steps are:

1. `GetLatestBlockhash` — transactions must commit to a recent
   blockhash; see [Chain Info Methods](Chain-Info-Methods).
2. `NewMessage` — compiles typed instructions into a deduplicated
   positional account layout; see [Message Builder](Message-Builder).
3. `Transaction.Sign` — fills per-signer signature slots in place;
   see [Transactions](Transactions) for the O(1) lookup details.
4. `SendTransaction` — returns the first-signature once the node
   accepts the transaction; see [Send Transaction](Send-Transaction).

## Want it confirmed, not just submitted?

`SendTransaction` only returns once the node accepts the bytes.
For a "send, wait for confirmation, retry on blockhash expiry"
flow, use the high-level helper:

```go
sig, err := c.SendAndConfirmTransaction(ctx,
    func(ctx context.Context, blockhash solana.Hash) (*solana.Transaction, error) {
        msg, _ := solana.NewMessage(payer.PublicKey(), instructions, blockhash)
        tx := solana.NewTransaction(*msg)
        if err := tx.Sign(ctx, payer); err != nil {
            return nil, err
        }
        return tx, nil
    },
    rpc.WithSendCommitment(solana.CommitmentConfirmed),
)
```

The builder is called once per send attempt, so blockhash refresh
is automatic. See [SendAndConfirmTransaction](SendAndConfirmTransaction).

## Next steps

- Understand the [transaction wire format](Transactions) and
  versioning.
- Learn the [call-option system](Call-Options) used by every RPC
  method.
- Stream live updates with the [WebSocket client](WebSocket-Client).
