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
	// repeated content should compress
	src := []byte("a long phrase here a long phrase here a long phrase here")
	enc, err := Encode(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) >= len(src) {
		t.Errorf("expected compression, got %d >= %d", len(enc), len(src))
	}
}

func TestDecodeErrors(t *testing.T) {
	// unterminated table entry
	_, err := Decode([]byte{0x00, 0x41})
	if err != ErrUnterminatedSeq {
		t.Errorf("expected ErrUnterminatedSeq, got %v", err)
	}

	// bad reference
	_, err = Decode([]byte{0x05})
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
