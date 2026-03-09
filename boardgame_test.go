package boardgame

import (
	"bytes"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	cases := []string{
		"hello hello hello",
		"if (a > b) a = b",
		"aaa bbb aaa bbb aaa bbb",
		"abcabc defdef abcabc",
		"no repeals here",
		"    ",
		" ",
	}
	for _, tc := range cases {
		src := []byte(tc)
		enc, err := Encode(src)
		if err != nil {
			t.Fatalf("Encode(%q): %v", tc, err)
		}
		dec, err := Decode(enc)
		if err != nil {
			t.Fatalf("Decode(Encode(%q)): %v", tc, err)
		}
		if !bytes.Equal(src, dec) {
			t.Errorf("round-trip failed for %q: got %q", tc, dec)
		}
	}
}

func TestCompressionSaves(t *testing.T) {
	src := []byte("a long phrase here a long phrase here a long phrase here")
	enc, err := Encode(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) >= len(src) {
		t.Errorf("expected compression, got %d >= %d", len(enc), len(src))
	}
}

func Test7BitPacking(t *testing.T) {
	// 8 ASCII bytes should pack into 7 bytes + 1 padding header = 8
	src := []byte("abcdefgh")
	packed := pack7(src)
	if len(packed) != 8 {
		t.Errorf("expected 8 ASCII chars to pack into 8 bytes (7+header), got %d", len(packed))
	}
	unpacked, err := unpack7(packed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src, unpacked) {
		t.Errorf("7-bit round-trip failed: got %q", unpacked)
	}
}

func Test7BitPackingSavesSpace(t *testing.T) {
	// any input should be ~12.5% smaller after 7-bit packing
	src := []byte("hello hello hello hello hello hello hello hello")
	packed := pack7(src)
	if len(packed) >= len(src) {
		t.Errorf("7-bit packing should save space: %d >= %d", len(packed), len(src))
	}
}

func TestDelEscape(t *testing.T) {
	// test that DEL escape preserves full 8-bit values
	src := []byte{0x41, 0xFF, 0x42} // A, 0xFF (needs escape), B
	packed := pack7(src)
	unpacked, err := unpack7(packed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src, unpacked) {
		t.Errorf("DEL escape round-trip failed: got %x, want %x", unpacked, src)
	}
}

func TestDecodeErrors(t *testing.T) {
	// bad reference in unpacked stream
	bad := pack7([]byte{0x05})
	_, err := Decode(bad)
	if err != ErrBadRef {
		t.Errorf("expected ErrBadRef, got %v", err)
	}
}

func TestEncodeRejectsInvalidBytes(t *testing.T) {
	_, err := Encode([]byte{0x00})
	if err != ErrByteOutOfRange {
		t.Errorf("expected ErrByteOutOfRange, got %v", err)
	}
	_, err = Encode([]byte{0x80})
	if err != ErrByteOutOfRange {
		t.Errorf("expected ErrByteOutOfRange, got %v", err)
	}
}

func TestEmptyInput(t *testing.T) {
	enc, err := Encode([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	dec, err := Decode(enc)
	if err != nil {
		t.Fatal(err)
	}
	if len(dec) != 0 {
		t.Errorf("expected empty, got %q", dec)
	}
}

func TestStats(t *testing.T) {
	orig := []byte("repeal repeal repeal repeal")
	enc, _ := Encode(orig)
	o, c, ratio := Stats(orig, enc)
	if o != len(orig) || c != len(enc) {
		t.Error("lengths mismatch")
	}
	if ratio <= 0 {
		t.Errorf("expected positive ratio, got %f", ratio)
	}
}
