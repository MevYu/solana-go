# IDL Codegen

**Anchor-IDL code generation is out of scope for this SDK.**

There is no first-party tool that consumes an Anchor IDL JSON
file and emits typed builders or account decoders. The decision
is deliberate:

- Every program builder in `programs/*/instructions.go` is
  hand-written next to the Solana runtime source, which makes
  the wire layout obvious to a code reviewer.
- An IDL-driven generator would couple the SDK to Anchor's
  schema, which is not the universal description of every
  Solana program (native programs, Pinocchio programs, and a
  growing fraction of dapps don't ship an IDL).
- Generated code in a Go module pulls all of the targeted
  programs into your build whether you call them or not,
  inflating binary size for SDK consumers.

## What to do instead

For Anchor programs you call, write the builder by hand using
the [encoding builder](Encoding-Builder) (or refer to
`programs/system/instructions.go` as a template). The Anchor
discriminator is `sha256("global:<method_name>")[:8]`:

```go
import (
    "crypto/sha256"
    "github.com/MevYu/solana-go/encoding"
)

var discTransfer = sha256.Sum256([]byte("global:transfer"))

func NewTransfer(authority, src, dst solana.PublicKey, amount uint64) solana.Instruction {
    return &myIx{
        accounts: []*solana.AccountMeta{
            solana.NewAccountMeta(authority, true, false),
            solana.NewAccountMeta(src, false, true),
            solana.NewAccountMeta(dst, false, true),
        },
        data: encoding.New().
            Raw(discTransfer[:8]).
            U64(amount).
            Bytes(),
    }
}
```

For account state, define a plain Go struct (no tags — the
decoder walks fields in declaration order) and decode with
`encoding.BorshDecodeTo(data, &v)`. Strip the 8-byte Anchor
discriminator first: `sha256("account:<TypeName>")[:8]`.

## Decoding program errors

Anchor's `Custom(u32)` error codes are decoded by
`rpc.DecodeTransactionError`; switch on the code as
documented in [Simulate With Decoded Errors](Simulate-With-Decoded-Errors).
You don't need a generator to match error codes — keep a small
hand-written constant table per program.

## Related

- [Encoding Builder](Encoding-Builder) — chained `encoding.New()`
  builder used by hand-written instruction builders.
- [Simulate With Decoded Errors](Simulate-With-Decoded-Errors) —
  matching `Custom(u32)` errors at runtime.
- Hand-written program builders for reference:
  [System Program](System-Program), [SPL Token Program](SPL-Token-Program),
  [Token-2022](Token-2022-Program).
