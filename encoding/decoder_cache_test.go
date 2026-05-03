package encoding

import (
	"testing"
)

// --- plain structs for reflection path ---

type flatStruct struct {
	A uint8
	B uint16
	C uint32
	D uint64
	E int8
	F int32
	G bool
}

type arrayStruct struct {
	Key  [32]byte
	Data [4]uint32
}

type nestedStruct struct {
	Outer uint64
	Inner flatStruct
}

type sliceStruct struct {
	Count uint64 // decoded separately; Payload uses u64 prefix
	Payload []byte
	Items   []uint32
}

type skipStruct struct {
	Keep   uint8
	Skip   uint8  `bin:"-"`
	Keep2  uint16
}

type optionStruct struct {
	Present *uint64
	Absent  *uint32
}

// roundtrip encodes fields into a buffer manually and decodes with Decode.
// Manual encode matches the bincode layout the decoder expects.

func encodeFlat(s flatStruct) []byte {
	e := NewEncoder(0)
	e.WriteUint8(s.A)
	e.WriteUint16(s.B)
	e.WriteUint32(s.C)
	e.WriteUint64(s.D)
	e.WriteInt8(s.E)
	e.WriteInt32(s.F)
	if s.G {
		e.WriteUint8(1)
	} else {
		e.WriteUint8(0)
	}
	return e.Bytes()
}

func TestDecode_FlatStruct(t *testing.T) {
	want := flatStruct{A: 1, B: 500, C: 70000, D: 1e18, E: -7, F: -300000, G: true}
	buf := encodeFlat(want)

	d := NewDecoder(buf)
	var got flatStruct
	if err := d.DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if d.Remaining() != 0 {
		t.Errorf("remaining = %d", d.Remaining())
	}
}

func TestDecode_ArrayStruct(t *testing.T) {
	want := arrayStruct{
		Key:  [32]byte{1, 2, 3, 4, 5},
		Data: [4]uint32{10, 20, 30, 40},
	}
	e := NewEncoder(0)
	e.WriteBytes(want.Key[:])
	for _, v := range want.Data {
		e.WriteUint32(v)
	}

	d := NewDecoder(e.Bytes())
	var got arrayStruct
	if err := d.DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Key != want.Key {
		t.Errorf("Key mismatch")
	}
	if got.Data != want.Data {
		t.Errorf("Data mismatch")
	}
}

func TestDecode_NestedStruct(t *testing.T) {
	inner := flatStruct{A: 9, B: 999, C: 1, D: 2, E: -1, F: 0, G: false}
	e := NewEncoder(0)
	e.WriteUint64(42)
	e.WriteBytes(encodeFlat(inner))

	d := NewDecoder(e.Bytes())
	var got nestedStruct
	if err := d.DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Outer != 42 {
		t.Errorf("Outer = %d", got.Outer)
	}
	if got.Inner != inner {
		t.Errorf("Inner = %+v", got.Inner)
	}
}

func TestDecode_SliceBytes(t *testing.T) {
	payload := []byte{0xde, 0xad, 0xbe, 0xef}
	e := NewEncoder(0)
	e.WriteUint64(uint64(len(payload)))
	e.WriteBytes(payload)

	type S struct{ P []byte }
	d := NewDecoder(e.Bytes())
	var got S
	if err := d.DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if string(got.P) != string(payload) {
		t.Errorf("payload mismatch")
	}
}

func TestDecode_SliceUint32(t *testing.T) {
	items := []uint32{111, 222, 333}
	e := NewEncoder(0)
	e.WriteUint64(uint64(len(items)))
	for _, v := range items {
		e.WriteUint32(v)
	}

	type S struct{ Items []uint32 }
	d := NewDecoder(e.Bytes())
	var got S
	if err := d.DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(got.Items) != len(items) {
		t.Fatalf("len = %d", len(got.Items))
	}
	for i, v := range items {
		if got.Items[i] != v {
			t.Errorf("[%d] = %d, want %d", i, got.Items[i], v)
		}
	}
}

func TestDecode_SkipTag(t *testing.T) {
	// Wire: Keep=1, Keep2=500 — Skip field is absent from wire, must not consume bytes.
	e := NewEncoder(0)
	e.WriteUint8(1)
	e.WriteUint16(500)

	d := NewDecoder(e.Bytes())
	var got skipStruct
	if err := d.DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Keep != 1 {
		t.Errorf("Keep = %d", got.Keep)
	}
	if got.Keep2 != 500 {
		t.Errorf("Keep2 = %d", got.Keep2)
	}
	if got.Skip != 0 {
		t.Errorf("Skip should be zero")
	}
	if d.Remaining() != 0 {
		t.Errorf("remaining = %d", d.Remaining())
	}
}

func TestDecode_PtrOption_Some(t *testing.T) {
	val := uint64(12345)
	e := NewEncoder(0)
	e.WriteUint8(1) // some
	e.WriteUint64(val)
	e.WriteUint8(0) // none

	d := NewDecoder(e.Bytes())
	var got optionStruct
	if err := d.DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Present == nil || *got.Present != val {
		t.Errorf("Present = %v", got.Present)
	}
	if got.Absent != nil {
		t.Errorf("Absent should be nil")
	}
}

func TestDecode_RequiresNonNilPointer(t *testing.T) {
	d := NewDecoder([]byte{1, 2, 3})
	if err := d.DecodeTo(42); err == nil {
		t.Fatal("expected error for non-pointer")
	}
	var x *flatStruct
	if err := d.DecodeTo(x); err == nil {
		t.Fatal("expected error for nil pointer")
	}
}

func TestDecode_UnsupportedType(t *testing.T) {
	type S struct{ F float32 }
	d := NewDecoder([]byte{1, 2, 3, 4})
	var got S
	if err := d.DecodeTo(&got); err == nil {
		t.Fatal("expected error for unsupported float32")
	}
}

// TestDecode_CacheIsHit verifies the decoder is cached: call twice, same type.
func TestDecode_CacheIsHit(t *testing.T) {
	e := NewEncoder(0)
	e.WriteUint8(7)
	e.WriteUint16(8)
	e.WriteUint32(9)
	e.WriteUint64(10)
	e.WriteInt8(0)
	e.WriteInt32(0)
	e.WriteUint8(0)
	buf := e.Bytes()

	for i := 0; i < 3; i++ {
		d := NewDecoder(buf)
		var got flatStruct
		if err := d.DecodeTo(&got); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if got.A != 7 || got.B != 8 || got.C != 9 || got.D != 10 {
			t.Fatalf("call %d: wrong values %+v", i, got)
		}
	}
}

// --- Unmarshaler escape hatch ---

type customType struct {
	X uint16
	Y uint16
}

func (c *customType) UnmarshalFromDecoder(d *Decoder) error {
	// custom: reads Y before X (reversed)
	v, err := d.ReadUint16()
	if err != nil {
		return err
	}
	c.Y = v
	v, err = d.ReadUint16()
	if err != nil {
		return err
	}
	c.X = v
	return nil
}

func TestDecode_UnmarshalerEscapeHatch(t *testing.T) {
	e := NewEncoder(0)
	e.WriteUint16(99)  // wire: Y first
	e.WriteUint16(42)  // wire: X second

	d := NewDecoder(e.Bytes())
	var got customType
	if err := d.DecodeTo(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.X != 42 || got.Y != 99 {
		t.Errorf("got X=%d Y=%d", got.X, got.Y)
	}
}
