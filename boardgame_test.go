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
	// 8 ASCII bytes = 56 bits = 7 packed bytes + 1 padding header = 8
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

func TestDelEscape8Bit(t *testing.T) {
	// DEL + high-bit byte in intermediate stream should round-trip
	// through pack7/unpack7 preserving both the DEL marker and raw byte
	src := []byte{0x41, delByte, 0xFF, 0x42}
	packed := pack7(src)
	unpacked, err := unpack7(packed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src, unpacked) {
		t.Errorf("DEL escape round-trip failed: got %x, want %x", unpacked, src)
	}
}

func TestSlotFreeAndReuse(t *testing.T) {
	// Build an intermediate stream by hand:
	// Define slot 1 = "ab", define slot 2 = "cd"
	// Use slot 1, free slot 1, define new entry "ef" (takes slot 1),
	// use new slot 1
	intermediate := []byte{
		0x00, 'a', 'b', 0x00, // define slot 1 = "ab"
		0x00, 'c', 'd', 0x00, // define slot 2 = "cd"
		0x01,             // ref slot 1 → "ab"
		0x02,             // ref slot 2 → "cd"
		0x00, 0x01,       // free slot 1
		0x00, 'e', 'f', 0x00, // define new entry → slot 1 = "ef"
		0x01,             // ref slot 1 → "ef"
	}
	got, err := tableExpand(intermediate)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("abcdef")
	if !bytes.Equal(got, want) {
		t.Errorf("slot free/reuse: got %q, want %q", got, want)
	}
}

func TestLowestFreeSlot(t *testing.T) {
	table := make(map[byte][]byte)
	// first slot should be 1
	if s := lowestFreeSlot(table); s != 1 {
		t.Errorf("empty table: got slot %d, want 1", s)
	}
	table[1] = []byte("a")
	table[2] = []byte("b")
	if s := lowestFreeSlot(table); s != 3 {
		t.Errorf("slots 1,2 taken: got slot %d, want 3", s)
	}
	// free slot 1
	delete(table, 1)
	if s := lowestFreeSlot(table); s != 1 {
		t.Errorf("slot 1 freed: got slot %d, want 1", s)
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
