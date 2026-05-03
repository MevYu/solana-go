package encoding_test

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/MevYu/solana-go/encoding"
)

// ----------------------------------------------------------------------------
// U128 / U256
// ----------------------------------------------------------------------------

func TestU128RoundTrip(t *testing.T) {
	// Rust: u128::MAX / 3 = 113427455640312821154458202477256070484
	want, _ := new(big.Int).SetString("113427455640312821154458202477256070484", 10)
	var u encoding.U128
	if err := u.SetBigInt(want); err != nil {
		t.Fatalf("SetBigInt: %v", err)
	}
	got := u.BigInt()
	if got.Cmp(want) != 0 {
		t.Fatalf("roundtrip: got %s want %s", got, want)
	}
	// Wire: serialise, deserialise, compare bytes.
	enc := encoding.NewEncoder(16)
	enc.WriteU128(u)
	if got := len(enc.Bytes()); got != 16 {
		t.Fatalf("encoded length %d, want 16", got)
	}
	dec := encoding.NewBinDecoder(enc.Bytes())
	back, err := dec.ReadU128()
	if err != nil {
		t.Fatalf("ReadU128: %v", err)
	}
	if back != u {
		t.Fatalf("decoded U128 mismatch: %x vs %x", back, u)
	}
}

func TestU128LittleEndian(t *testing.T) {
	// 1 as a u128 serialises as 0x01 followed by 15 zero bytes.
	var one encoding.U128
	one.SetUint64(1)
	enc := encoding.NewEncoder(16)
	enc.WriteU128(one)
	want := make([]byte, 16)
	want[0] = 1
	if !bytes.Equal(enc.Bytes(), want) {
		t.Fatalf("little-endian: got %x, want %x", enc.Bytes(), want)
	}
}

func TestU128JSON(t *testing.T) {
	var u encoding.U128
	u.SetUint64(42)
	b, err := u.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(b) != `"42"` {
		t.Fatalf("MarshalJSON: got %s want %q", b, `"42"`)
	}
	var got encoding.U128
	if err := got.UnmarshalJSON([]byte(`"42"`)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if got != u {
		t.Fatalf("JSON round trip: %v != %v", got, u)
	}
}

func TestU256RoundTrip(t *testing.T) {
	want, _ := new(big.Int).SetString(
		"57896044618658097711785492504343953926634992332820282019728792003956564819967", 10,
	) // (2^255 - 1)
	var u encoding.U256
	if err := u.SetBigInt(want); err != nil {
		t.Fatalf("SetBigInt: %v", err)
	}
	if got := u.BigInt(); got.Cmp(want) != 0 {
		t.Fatalf("roundtrip: got %s want %s", got, want)
	}
	enc := encoding.NewEncoder(32)
	enc.WriteU256(u)
	dec := encoding.NewBinDecoder(enc.Bytes())
	back, err := dec.ReadU256()
	if err != nil {
		t.Fatalf("ReadU256: %v", err)
	}
	if back != u {
		t.Fatalf("decoded U256 mismatch")
	}
}

func TestU128OverflowRejected(t *testing.T) {
	x := new(big.Int).Lsh(big.NewInt(1), 129)
	var u encoding.U128
	if err := u.SetBigInt(x); err == nil {
		t.Fatal("expected overflow error for 129-bit input, got nil")
	}
	neg := big.NewInt(-1)
	if err := u.SetBigInt(neg); err == nil {
		t.Fatal("expected negative error, got nil")
	}
}

// ----------------------------------------------------------------------------
// Plan-cached Decode — structs, slices, arrays, options, registered types
// ----------------------------------------------------------------------------

type primStruct struct {
	A uint8
	B uint16
	C uint32
	D uint64
	E int32
	F bool
}

func TestDecodePrimStruct(t *testing.T) {
	enc := encoding.NewEncoder(32)
	enc.WriteUint8(0xab)
	enc.WriteUint16(0xcdef)
	enc.WriteUint32(0x12345678)
	enc.WriteUint64(0x1122334455667788)
	enc.WriteInt32(-1)
	enc.WriteUint8(1)
	var got primStruct
	if err := encoding.NewBinDecoder(enc.Bytes()).DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want := primStruct{0xab, 0xcdef, 0x12345678, 0x1122334455667788, -1, true}
	if got != want {
		t.Fatalf("primStruct: got %+v want %+v", got, want)
	}
}

type withU128 struct {
	Flag uint8
	Big  encoding.U128
	Huge encoding.U256
}

func TestDecodeU128Field(t *testing.T) {
	var lo encoding.U128
	lo.SetLoHi(0x1111111111111111, 0x2222222222222222)
	var huge encoding.U256
	for i := range huge {
		huge[i] = byte(i + 1)
	}
	enc := encoding.NewEncoder(64)
	enc.WriteUint8(7)
	enc.WriteU128(lo)
	enc.WriteU256(huge)
	var got withU128
	if err := encoding.NewBinDecoder(enc.Bytes()).DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Flag != 7 || got.Big != lo || got.Huge != huge {
		t.Fatalf("U128 field decode mismatch: %+v", got)
	}
}

type withOption struct {
	Header uint32
	Inner  *innerOpt
}

type innerOpt struct {
	X uint64
	Y uint16
}

func TestDecodeOptionSome(t *testing.T) {
	enc := encoding.NewEncoder(16)
	enc.WriteUint32(42)
	enc.WriteUint8(1) // Some
	enc.WriteUint64(0xdeadbeef)
	enc.WriteUint16(7)
	var got withOption
	if err := encoding.NewBinDecoder(enc.Bytes()).DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Header != 42 || got.Inner == nil || got.Inner.X != 0xdeadbeef || got.Inner.Y != 7 {
		t.Fatalf("Some decode mismatch: %+v", got)
	}
}

func TestDecodeOptionNone(t *testing.T) {
	enc := encoding.NewEncoder(8)
	enc.WriteUint32(99)
	enc.WriteUint8(0) // None
	var got withOption
	if err := encoding.NewBinDecoder(enc.Bytes()).DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Header != 99 || got.Inner != nil {
		t.Fatalf("None decode mismatch: %+v", got)
	}
}

type withByteSlice struct {
	Len  uint8
	Body []byte
	Tail uint16
}

func TestDecodeByteSlice(t *testing.T) {
	body := []byte("hello world")
	enc := encoding.NewEncoder(32)
	enc.WriteUint8(0xaa)
	enc.WriteUint64(uint64(len(body))) // default bincode prefix
	enc.WriteBytes(body)
	enc.WriteUint16(0xbeef)
	var got withByteSlice
	if err := encoding.NewBinDecoder(enc.Bytes()).DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Len != 0xaa || string(got.Body) != string(body) || got.Tail != 0xbeef {
		t.Fatalf("byte-slice decode mismatch: %+v", got)
	}
}

type elem struct {
	Key uint32
	Val uint64
}

type withElemSlice struct {
	Count uint8
	Items []elem
	Marker uint16
}

func TestDecodeStructSlice(t *testing.T) {
	enc := encoding.NewEncoder(64)
	enc.WriteUint8(0xa0)
	enc.WriteUint64(3) // slice len
	for i := uint64(1); i <= 3; i++ {
		enc.WriteUint32(uint32(i))
		enc.WriteUint64(i * 100)
	}
	enc.WriteUint16(0x1234)
	var got withElemSlice
	if err := encoding.NewBinDecoder(enc.Bytes()).DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Count != 0xa0 || got.Marker != 0x1234 || len(got.Items) != 3 {
		t.Fatalf("struct-slice outer mismatch: %+v", got)
	}
	for i, e := range got.Items {
		want := elem{uint32(i + 1), uint64(i+1) * 100}
		if e != want {
			t.Fatalf("element %d: got %+v want %+v", i, e, want)
		}
	}
}

type fixedArr struct {
	Tag  uint8
	Keys [4]uint64
}

func TestDecodeFixedArray(t *testing.T) {
	enc := encoding.NewEncoder(64)
	enc.WriteUint8(0x55)
	for i := uint64(0); i < 4; i++ {
		enc.WriteUint64(i + 1)
	}
	var got fixedArr
	if err := encoding.NewBinDecoder(enc.Bytes()).DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Tag != 0x55 || got.Keys != [4]uint64{1, 2, 3, 4} {
		t.Fatalf("fixed-array decode mismatch: %+v", got)
	}
}

// Registered decoder short-circuits the reflective walk even when the type
// appears as a nested field.
type regTarget struct {
	N uint32
}

func init() {
	encoding.RegisterDecoder[regTarget](func(d *encoding.Decoder, p *regTarget) error {
		v, err := d.ReadUint32()
		if err != nil {
			return err
		}
		// Sentinel: prove the hand-written path ran by inverting the value.
		p.N = ^v
		return nil
	})
}

type outerReg struct {
	Prefix uint8
	R      regTarget
}

func TestRegisteredDecoderTakesPrecedence(t *testing.T) {
	enc := encoding.NewEncoder(8)
	enc.WriteUint8(0x11)
	enc.WriteUint32(0xAAAAAAAA)
	var got outerReg
	if err := encoding.NewBinDecoder(enc.Bytes()).DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Prefix != 0x11 || got.R.N != ^uint32(0xAAAAAAAA) {
		t.Fatalf("registered decoder not called: %+v", got)
	}
}

// Unmarshaler interface dispatch without registry registration.
type ifaceTarget struct {
	marker uint64
}

func (t *ifaceTarget) UnmarshalFromDecoder(d *encoding.Decoder) error {
	v, err := d.ReadUint64()
	if err != nil {
		return err
	}
	t.marker = v ^ 0xFFFFFFFFFFFFFFFF
	return nil
}

type outerIface struct {
	Tag uint8
	T   ifaceTarget
}

func TestUnmarshalerInterfaceDispatch(t *testing.T) {
	enc := encoding.NewEncoder(16)
	enc.WriteUint8(1)
	enc.WriteUint64(0x1234567890ABCDEF)
	var got outerIface
	if err := encoding.NewBinDecoder(enc.Bytes()).DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Tag != 1 || got.T.marker != (0x1234567890ABCDEF^0xFFFFFFFFFFFFFFFF) {
		t.Fatalf("iface dispatch not invoked: %+v", got)
	}
}

// Short-buffer errors propagate out through the plan executor.
func TestDecodeShortBuffer(t *testing.T) {
	var got withElemSlice
	err := encoding.NewBinDecoder([]byte{0x01}).DecodeTo(&got)
	if err == nil {
		t.Fatal("expected error on truncated input")
	}
	if !errors.Is(err, encoding.ErrShortBuffer) {
		t.Fatalf("expected ErrShortBuffer, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// Vec<Entry> decoding — the workload the user actually cares about
// ----------------------------------------------------------------------------

// simplifiedTx is a stand-in for the full Transaction type used to exercise
// the Vec<Vec<T>> pattern without dragging programs/... into the encoding
// test binary. Fields are typical transaction-ish things.
type simplifiedTx struct {
	NumSigs uint8
	Lamports uint64
	Memo    string
}

type entry struct {
	NumHashes uint64
	Hash      [32]byte
	Txs       []simplifiedTx
}

func TestDecodeVecOfEntries(t *testing.T) {
	enc := encoding.NewEncoder(256)
	enc.WriteUint64(2) // 2 entries
	// entry 0
	enc.WriteUint64(0x1000)
	for i := 0; i < 32; i++ {
		enc.WriteUint8(byte(i))
	}
	enc.WriteUint64(2) // 2 transactions
	// tx 0.0
	enc.WriteUint8(1)
	enc.WriteUint64(5000)
	msg1 := "hello"
	enc.WriteUint64(uint64(len(msg1)))
	enc.WriteBytes([]byte(msg1))
	// tx 0.1
	enc.WriteUint8(2)
	enc.WriteUint64(9999)
	msg2 := "world"
	enc.WriteUint64(uint64(len(msg2)))
	enc.WriteBytes([]byte(msg2))
	// entry 1
	enc.WriteUint64(0x2000)
	for i := 0; i < 32; i++ {
		enc.WriteUint8(byte(0xff - i))
	}
	enc.WriteUint64(1)
	enc.WriteUint8(3)
	enc.WriteUint64(1)
	msg3 := "!"
	enc.WriteUint64(uint64(len(msg3)))
	enc.WriteBytes([]byte(msg3))

	var entries []entry
	if err := encoding.NewBinDecoder(enc.Bytes()).DecodeTo(&entries); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	if entries[0].NumHashes != 0x1000 || entries[1].NumHashes != 0x2000 {
		t.Fatalf("NumHashes wrong: %x / %x", entries[0].NumHashes, entries[1].NumHashes)
	}
	if len(entries[0].Txs) != 2 || len(entries[1].Txs) != 1 {
		t.Fatalf("tx counts wrong: %d / %d", len(entries[0].Txs), len(entries[1].Txs))
	}
	if entries[0].Txs[1].Memo != "world" || entries[1].Txs[0].Memo != "!" {
		t.Fatalf("memo decode mismatch: %+v", entries)
	}
}
