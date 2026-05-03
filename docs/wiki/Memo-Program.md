# Memo Program

The **Memo Program** records a UTF-8 string in the transaction
log. It performs no other side effect — there is no state, no
fee beyond the per-signature cost, no return data. Use it to
attach an audit trail or correlation id to an on-chain
transaction.

```go
import "github.com/MevYu/solana-go/programs/memo"
```

## Program id

```go
var memo.ProgramID = solana.MustPublicKey(
    "MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr")
```

This is the v2 program (with optional signers). The v1 program
at `Memo1UhkJRfHyvLMcVucJwxXeuD728EqVDDwQDxFMNo` is deprecated.

## API

```go
func Log(memo string, signers ...*solana.AccountMeta) solana.Instruction
```

The instruction's data is the raw UTF-8 bytes of `memo` with no
length prefix and no discriminator — the memo program treats
its entire data buffer as the message.

`signers` is an optional list of accounts that must sign the
transaction; the memo program enforces signer-presence on every
account passed to it. Pass `nil` for an unsigned memo.

```go
ix := memo.Log("order-12345 from session 9b3f", nil)
```

If you want one or more wallets to attest to the memo, pass them:

```go
attest := solana.NewAccountMeta(payer.PublicKey(), true, false)
ix := memo.Log("payment authorised", attest)
```

## Notes

- There is no length limit imposed by the SDK; the on-chain
  program processes whatever bytes you provide. Transactions
  are still bounded by the 1232-byte packet size.
- Memos are visible to anyone reading the transaction log;
  do not put secrets in them.
- `rpc.DecodeTransactionError` does not need to special-case
  memo errors — the program returns `Custom(0)` on a non-UTF-8
  payload, which surfaces as a normal `*InstructionError`.

## Related

- [Send Transaction](Send-Transaction) — wrap the memo
  instruction in a transaction the way you would any other.
