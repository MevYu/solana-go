# Program-Derived Addresses

A **Program-Derived Address** (PDA) is a 32-byte key that is not
an Ed25519 public key: it does not lie on the ed25519 curve and
therefore has no corresponding private key. A program owns a PDA
by virtue of having derived it; the Solana runtime validates this
ownership at execution time.

PDAs are what let a program "own" mutable state without a user
having to co-sign every access: the program can sign CPI calls on
behalf of the PDA using `invoke_signed`, and no-one else can.

## The derivation

```
sha256(seed1 || seed2 || … || programID || "ProgramDerivedAddress")
```

If the resulting 32 bytes happen to lie on the ed25519 curve,
the derivation is invalid and callers must vary the seeds. In
practice this is handled by appending a single **bump seed** and
iterating from 255 down until an off-curve hash is found.

## `CreateProgramAddress`

```go
func CreateProgramAddress(seeds [][]byte, programID PublicKey) (PublicKey, error)
```

Direct counterpart of the Solana runtime's
`create_program_address`. It hashes the seeds in order, checks
the result is off-curve, and returns it. It returns an error if:

- you pass more than `MaxPDASeeds` (16) seeds
- any seed is longer than `MaxPDASeedLength` (32 bytes)
- the computed hash lies on the curve (you must vary the seeds)

Use this when you already know the bump and want to re-derive
the PDA without the search loop.

## `FindProgramAddress`

```go
func FindProgramAddress(seeds [][]byte, programID PublicKey) (PublicKey, uint8, error)
```

Iterates bump seeds from 255 down to 0 until it finds one that
produces an off-curve hash. Returns the PDA and the bump that
worked. Store the bump alongside the PDA if you plan to re-derive
the same address later — re-deriving without the stored bump
wastes CPU by re-running the search.

`FindProgramAddress` returns an error only if the seed list would
already exceed `MaxPDASeeds` after appending the bump, or in the
astronomically unlikely event that every bump from 255 down to 0
produces an on-curve point (which would require ~2^256 of bad
luck).

## Example: derive an ATA address

The Associated Token Account program's PDA is derived with the
seed order `[wallet, tokenProgram, mint]`. This is exactly what
`ata.FindAssociatedTokenAddress` does:

```go
seeds := [][]byte{
    wallet[:],
    tokenProgram[:],
    mint[:],
}
ata, bump, err := solana.FindProgramAddress(seeds, ata.ProgramID)
```

See [Associated Token Account](Associated-Token-Account-Program)
for the typed builder that wraps this.

## Example: derive an Address Lookup Table PDA

```go
slotBytes := make([]byte, 8)
binary.LittleEndian.PutUint64(slotBytes, recentSlot)
seeds := [][]byte{authority[:], slotBytes}
table, bump, err := solana.FindProgramAddress(seeds, addresslookuptable.ProgramID)
```

See [Address Lookup Tables](Address-Lookup-Tables) and
`addresslookuptable.DeriveLookupTableAddress` for the packaged form.

## Off-curve check

The check is `edwards25519.Point.SetBytes`: a real Ed25519
public key decompresses successfully, a PDA does not. The SDK
uses `filippo.io/edwards25519` for this — a transitive dependency
pulled in only via `pda.go`.

## Why this matters for security

The PDA derivation is **deterministic** and **collision-resistant**:
anybody can re-derive a PDA from its seeds and program id, so
callers can verify that an account they received was in fact
produced by the program they trust. This is the foundation of
programs like ATA, Metaplex metadata, Squads multisig, and so on.

## Related

- [System Program](System-Program) — `NewCreateAccount` is what
  a program uses to fund a PDA after deriving it.
- [Associated Token Account](Associated-Token-Account-Program) —
  the most common real-world PDA.
- [Address Lookup Tables](Address-Lookup-Tables) — another PDA
  with a slot-based seed.
