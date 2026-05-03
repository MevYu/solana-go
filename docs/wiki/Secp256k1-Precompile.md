# Secp256k1 Precompile

The **secp256k1 precompile** is a native program at
`KeccakSecp256k11111111111111111111111111111` that verifies
ECDSA signatures over keccak-256 hashes — the same scheme used
by Ethereum. Using it lets a Solana program defer signature
verification to the runtime instead of running it inside a BPF
program.

```go
import "github.com/MevYu/solana-go/programs/secp256k1"
```

The on-chain program does not expose typed sub-instructions: its
single instruction is a self-describing byte buffer that
encodes one or more signature records. A signature record names
the offsets where the signing program (typically a
co-instruction in the same transaction) wrote the
`(eth_address, signature, recovery_id, message)` tuple, and
the runtime walks those offsets to verify.

## Builders

### `NewVerifyEthSignature`

```go
func NewVerifyEthSignature(
    ethAddress [20]byte,
    signature [64]byte,
    recoveryID uint8,
    message []byte,
) (solana.Instruction, error)
```

Convenience for the single-record case: the SDK lays out the
header, signature offsets table, and the
`(eth_address, signature, recovery_id, message)` payload into
the instruction data, and points the offsets at the right
bytes within the buffer.

`ethAddress` is `keccak256(uncompressed_pubkey)[12:]` (the last
20 bytes), matching the standard Ethereum address derivation.

### `NewSignatureVerifyInstruction`

```go
type Secp256k1SignatureOffsets struct {
    SignatureOffset            uint16
    SignatureInstructionIndex  uint8
    EthAddressOffset           uint16
    EthAddressInstructionIndex uint8
    MessageDataOffset          uint16
    MessageDataSize            uint16
    MessageInstructionIndex    uint8
}

func NewSignatureVerifyInstruction(signatures []Secp256k1SignatureOffsets) (solana.Instruction, error)
```

Lower-level entry point for transactions where the signature
data lives in a separate instruction (`*InstructionIndex` = the
ix slot inside the same transaction; `0xFF` means "look in this
instruction's own data"). Use this when bundling many
signatures or when paying signing data via instruction-level
deduplication.

### `NewRawInstruction` / `NewInstruction`

```go
func NewRawInstruction(data []byte) solana.Instruction
func NewInstruction(data []byte) solana.Instruction
```

Escape hatches that emit an instruction targeting the
secp256k1 precompile with caller-supplied data. Use only if
you have already constructed the offsets table by hand.

## Layout note

The SDK lays the single-record buffer out as
`signature | recovery_id | eth_address | message` — slightly
different from `solana_sdk::secp256k1_instruction::new_secp256k1_instruction`,
which uses `eth_address | signature | recovery_id | message`.
Both forms are wire-compatible because every field is referenced
through its offset in the header table; only golden-bytes
comparisons against the Rust SDK's exact layout differ.

## Related

- [Send Transaction](Send-Transaction) — bundle the verify
  instruction alongside the program that consumes its result.
