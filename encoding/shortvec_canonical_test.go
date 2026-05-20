package encoding

import (
	"errors"
	"testing"
)

// Solana's consensus shortvec decoder enforces minimal (canonical)
// encoding. DecodeShortvec must reject overlong forms a validator rejects.
func TestDecodeShortvec_RejectsNonCanonical(t *testing.T) {
	overlong := [][]byte{
		{0x80, 0x00},       // value 0 in 2 bytes (canonical: {0x00})
		{0x81, 0x00},       // value 1 in 2 bytes (canonical: {0x01})
		{0x80, 0x80, 0x00}, // value 0 in 3 bytes
		{0xff, 0xff, 0x00}, // 2-byte value in 3 bytes
	}
	for _, b := range overlong {
		if _, _, err := DecodeShortvec(b); !errors.Is(err, ErrInvalidShortvec) {
			t.Errorf("DecodeShortvec(%x) err = %v, want ErrInvalidShortvec", b, err)
		}
	}
}

func TestDecodeShortvec_AcceptsCanonical(t *testing.T) {
	cases := []struct {
		in   []byte
		want uint16
		n    int
	}{
		{[]byte{0x00}, 0, 1},
		{[]byte{0x7f}, 127, 1},
		{[]byte{0x80, 0x01}, 128, 2},
		{[]byte{0xff, 0x7f}, 16383, 2},
		{[]byte{0x80, 0x80, 0x01}, 16384, 3},
		{[]byte{0xff, 0xff, 0x03}, 65535, 3},
	}
	for _, c := range cases {
		v, n, err := DecodeShortvec(c.in)
		if err != nil || v != c.want || n != c.n {
			t.Errorf("DecodeShortvec(%x) = (%d,%d,%v), want (%d,%d,nil)", c.in, v, n, err, c.want, c.n)
		}
	}
}
