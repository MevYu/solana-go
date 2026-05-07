package solana

import (
	"testing"
	"unsafe"

	"github.com/MevYu/solana-go/encoding"
)

// Bench results — arm64, GOMAXPROCS=20, 10 runs each, mean ns/op:
//
//                          reflective    Unmarshaler    speedup
//   PoolState (seq)            350.5         334.5       −4.6%
//   PoolStateParallel          220.1         185.4      −15.8%
//   PoolStateHighParallel      199.9         186.0       −7.0%
//
// Allocations identical (2/op). The Unmarshaler win comes from collapsing
// 30 op-switch dispatches into one closure; the reflective baseline still
// benefits from decodePlanCache (compile-once-per-type), so this isn't a
// "reflection vs no reflection" comparison — it's "30 ops vs 1 op".
//
// Repro: go test -run=^$ -bench='BenchmarkDecodePoolState' -benchmem -count=10 .

// poolState mimics a realistic AMM-pool state struct (Raydium-CLMM /
// Whirlpool / Meteora shape): 8-byte discriminator, several PublicKey
// references (mints, vaults, owner, oracle), two U128 fields (sqrt-price,
// liquidity), and a long tail of u64 pool parameters and accumulators.
type poolState struct {
	Discriminator [8]byte

	Bump   uint8
	Status uint8

	Owner     PublicKey
	Authority PublicKey

	MintA     PublicKey
	VaultA    PublicKey
	DecimalsA uint8

	MintB     PublicKey
	VaultB    PublicKey
	DecimalsB uint8

	Observation PublicKey

	SqrtPrice   U128
	Liquidity   U128
	TickCurrent int32
	TickSpacing uint16

	FeeGrowthA      uint64
	FeeGrowthB      uint64
	FeeRate         uint32
	ProtocolFeeRate uint32

	TotalLPSupply uint64
	VolumeA       uint64
	VolumeB       uint64
	TWAPSqrtPrice uint64

	RewardMint0      PublicKey
	RewardEmissions0 uint64
	RewardMint1      PublicKey
	RewardEmissions1 uint64

	LastUpdateSlot uint64
	OpenTimestamp  uint64
}

// poolStateHand has identical wire layout to poolState; we keep them as
// distinct Go types so the Unmarshaler dispatch can target one without
// affecting the other (the plan compiler keys interface satisfaction off
// reflect.Type).
type poolStateHand poolState

// UnmarshalFromDecoder is the hand-written single-shot decoder for
// poolStateHand. It reads every field directly via Decoder methods —
// no reflection, no op-switch dispatch. This is what implementing
// Unmarshaler buys you when the receiver is an account-shaped struct.
func (p *poolStateHand) UnmarshalFromDecoder(d *encoding.Decoder) error {
	// Discriminator [8]byte
	b, err := d.ReadBytes(8)
	if err != nil {
		return err
	}
	copy(p.Discriminator[:], b)

	if p.Bump, err = d.ReadUint8(); err != nil {
		return err
	}
	if p.Status, err = d.ReadUint8(); err != nil {
		return err
	}

	// Owner / Authority
	if b, err = d.ReadBytes(PublicKeySize); err != nil {
		return err
	}
	p.Owner = *(*PublicKey)(unsafe.Pointer(&b[0]))
	if b, err = d.ReadBytes(PublicKeySize); err != nil {
		return err
	}
	p.Authority = *(*PublicKey)(unsafe.Pointer(&b[0]))

	// MintA / VaultA / DecimalsA
	if b, err = d.ReadBytes(PublicKeySize); err != nil {
		return err
	}
	p.MintA = *(*PublicKey)(unsafe.Pointer(&b[0]))
	if b, err = d.ReadBytes(PublicKeySize); err != nil {
		return err
	}
	p.VaultA = *(*PublicKey)(unsafe.Pointer(&b[0]))
	if p.DecimalsA, err = d.ReadUint8(); err != nil {
		return err
	}

	// MintB / VaultB / DecimalsB
	if b, err = d.ReadBytes(PublicKeySize); err != nil {
		return err
	}
	p.MintB = *(*PublicKey)(unsafe.Pointer(&b[0]))
	if b, err = d.ReadBytes(PublicKeySize); err != nil {
		return err
	}
	p.VaultB = *(*PublicKey)(unsafe.Pointer(&b[0]))
	if p.DecimalsB, err = d.ReadUint8(); err != nil {
		return err
	}

	// Observation
	if b, err = d.ReadBytes(PublicKeySize); err != nil {
		return err
	}
	p.Observation = *(*PublicKey)(unsafe.Pointer(&b[0]))

	// SqrtPrice / Liquidity (U128 = [16]byte)
	if b, err = d.ReadBytes(16); err != nil {
		return err
	}
	p.SqrtPrice = *(*U128)(unsafe.Pointer(&b[0]))
	if b, err = d.ReadBytes(16); err != nil {
		return err
	}
	p.Liquidity = *(*U128)(unsafe.Pointer(&b[0]))

	if p.TickCurrent, err = d.ReadInt32(); err != nil {
		return err
	}
	if p.TickSpacing, err = d.ReadUint16(); err != nil {
		return err
	}

	if p.FeeGrowthA, err = d.ReadUint64(); err != nil {
		return err
	}
	if p.FeeGrowthB, err = d.ReadUint64(); err != nil {
		return err
	}
	if p.FeeRate, err = d.ReadUint32(); err != nil {
		return err
	}
	if p.ProtocolFeeRate, err = d.ReadUint32(); err != nil {
		return err
	}

	if p.TotalLPSupply, err = d.ReadUint64(); err != nil {
		return err
	}
	if p.VolumeA, err = d.ReadUint64(); err != nil {
		return err
	}
	if p.VolumeB, err = d.ReadUint64(); err != nil {
		return err
	}
	if p.TWAPSqrtPrice, err = d.ReadUint64(); err != nil {
		return err
	}

	// Reward 0
	if b, err = d.ReadBytes(PublicKeySize); err != nil {
		return err
	}
	p.RewardMint0 = *(*PublicKey)(unsafe.Pointer(&b[0]))
	if p.RewardEmissions0, err = d.ReadUint64(); err != nil {
		return err
	}
	// Reward 1
	if b, err = d.ReadBytes(PublicKeySize); err != nil {
		return err
	}
	p.RewardMint1 = *(*PublicKey)(unsafe.Pointer(&b[0]))
	if p.RewardEmissions1, err = d.ReadUint64(); err != nil {
		return err
	}

	if p.LastUpdateSlot, err = d.ReadUint64(); err != nil {
		return err
	}
	if p.OpenTimestamp, err = d.ReadUint64(); err != nil {
		return err
	}
	return nil
}

func makePoolStatePayload() []byte {
	e := encoding.NewEncoder(400)
	for i := 0; i < 8; i++ {
		e.WriteUint8(byte(0xA0 + i))
	}
	e.WriteUint8(255) // bump
	e.WriteUint8(1)   // status
	for k := 0; k < 2; k++ {
		var pk PublicKey
		pk[0] = byte(0x10 + k)
		e.WriteBytes(pk[:])
	}
	for k := 0; k < 2; k++ {
		var pk PublicKey
		pk[0] = byte(0x20 + k)
		e.WriteBytes(pk[:])
	}
	e.WriteUint8(6)
	for k := 0; k < 2; k++ {
		var pk PublicKey
		pk[0] = byte(0x30 + k)
		e.WriteBytes(pk[:])
	}
	e.WriteUint8(9)
	var obs PublicKey
	obs[0] = 0x40
	e.WriteBytes(obs[:])
	var sqrtPrice U128
	sqrtPrice.SetLoHi(0xDEADBEEFCAFEBABE, 0)
	e.WriteU128(sqrtPrice)
	var liquidity U128
	liquidity.SetUint64(1_000_000_000)
	e.WriteU128(liquidity)
	e.WriteInt32(-1234)
	e.WriteUint16(60)
	e.WriteUint64(111)
	e.WriteUint64(222)
	e.WriteUint32(2500)
	e.WriteUint32(500)
	e.WriteUint64(1_000_000)
	e.WriteUint64(50_000)
	e.WriteUint64(60_000)
	e.WriteUint64(0xFFFF_FFFF_FFFF_FFFF)
	for k := 0; k < 2; k++ {
		var pk PublicKey
		pk[0] = byte(0x50 + k)
		e.WriteBytes(pk[:])
		e.WriteUint64(uint64(1000 * (k + 1)))
	}
	e.WriteUint64(123_456_789)
	e.WriteUint64(987_654_321)
	return e.Bytes()
}

// Reflective path (poolState NOT registered).

func BenchmarkDecodePoolState(b *testing.B) {
	data := makePoolStatePayload()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v poolState
		if err := encoding.BinDecodeTo(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodePoolStateParallel(b *testing.B) {
	data := makePoolStatePayload()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var v poolState
			if err := encoding.BinDecodeTo(data, &v); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkDecodePoolStateHighParallel(b *testing.B) {
	data := makePoolStatePayload()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.SetParallelism(10)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var v poolState
			if err := encoding.BinDecodeTo(data, &v); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Hand-written + Unmarshaler interface path.

func BenchmarkDecodePoolStateHand(b *testing.B) {
	data := makePoolStatePayload()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v poolStateHand
		if err := encoding.BinDecodeTo(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodePoolStateHandParallel(b *testing.B) {
	data := makePoolStatePayload()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var v poolStateHand
			if err := encoding.BinDecodeTo(data, &v); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkDecodePoolStateHandHighParallel(b *testing.B) {
	data := makePoolStatePayload()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.SetParallelism(10)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var v poolStateHand
			if err := encoding.BinDecodeTo(data, &v); err != nil {
				b.Fatal(err)
			}
		}
	})
}
