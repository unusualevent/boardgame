package boardgame

import (
	"bytes"
	"strings"
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
		"`backtick` `backtick` `backtick`",
		"func main() {\n\tfmt.Println(`hello`)\n}",
		"~!@#$%^&*()_+-=[]{}|;':\",./<>?",
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

func TestExtendedRef(t *testing.T) {
	// Build intermediate stream that fills all direct slots (skipping
	// reserved bytes 0x09/0x0A), then defines one more entry which
	// must land in the extended range and be referenced via null-DEL-byte.
	intermediate := make([]byte, 0)

	// define entries for all non-reserved direct slots
	for s := byte(1); s <= maxDirectRef; s++ {
		if isReserved(s) {
			continue
		}
		seq := []byte{minGlyph + s, minGlyph + s}
		intermediate = append(intermediate, 0x00)
		intermediate = append(intermediate, seq...)
		intermediate = append(intermediate, 0x00)
	}
	// next free slot is 0x1A (since 0x09/0x0A were skipped, not filled)
	intermediate = append(intermediate, 0x00, 'h', 'i', 0x00)
	// reference it via null-DEL-0x1A
	intermediate = append(intermediate, 0x00, delByte, 0x1A)

	got, err := tableExpand(intermediate)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("hi")) {
		t.Errorf("extended ref: got %q, want %q", got, "hi")
	}
}

func TestExtendedRefManual(t *testing.T) {
	// Test null-DEL-byte with an arbitrary slot like 0xF0.
	// Fill all non-reserved direct slots, then extended slots up to 0xEF.
	intermediate := []byte{
		0x00, 'a', 'b', 0x00, // slot 1 = "ab"
	}
	// fill remaining non-reserved direct slots (2–25, skipping reserved)
	for s := byte(2); s <= maxDirectRef; s++ {
		if isReserved(s) {
			continue
		}
		intermediate = append(intermediate, 0x00, minGlyph+s, minGlyph+s, 0x00)
	}
	// slots 0x1A through 0xEF
	for s := byte(0x1A); s <= 0xEF; s++ {
		intermediate = append(intermediate, 0x00, minGlyph+(s%84), minGlyph+((s+1)%84), 0x00)
	}
	// slot 0xF0 = "cd"
	intermediate = append(intermediate, 0x00, 'c', 'd', 0x00)
	// direct ref slot 1, then extended ref slot 0xF0
	intermediate = append(intermediate, 0x01, 0x00, delByte, 0xF0)

	got, err := tableExpand(intermediate)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("abcd")) {
		t.Errorf("extended ref 0xF0: got %q, want %q", got, "abcd")
	}
}

func TestRefCost(t *testing.T) {
	if rc := refCost(0x01); rc != 1 {
		t.Errorf("refCost(0x01) = %d, want 1", rc)
	}
	if rc := refCost(maxDirectRef); rc != 1 {
		t.Errorf("refCost(0x19) = %d, want 1", rc)
	}
	if rc := refCost(0x1A); rc != 3 {
		t.Errorf("refCost(0x1A) = %d, want 3", rc)
	}
	if rc := refCost(0xF0); rc != 3 {
		t.Errorf("refCost(0xF0) = %d, want 3", rc)
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

func TestFullPrintableASCII(t *testing.T) {
	// Every valid literal byte should round-trip.
	var buf []byte
	buf = append(buf, tab, newline)
	for b := byte(minGlyph); b <= maxGlyph; b++ {
		buf = append(buf, b)
	}
	// Repeat so there's something to compress.
	src := bytes.Repeat(buf, 3)
	enc, err := Encode(src)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	dec, err := Decode(enc)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if !bytes.Equal(src, dec) {
		t.Error("round-trip failed for full printable ASCII set")
	}
}

// makeSource generates a pseudo-source-code string of approximately n bytes.
func makeSource(n int) []byte {
	lines := []string{
		"func main() {",
		"\tfmt.Println(`hello world`)",
		"\tfor i := 0; i < 100; i++ {",
		"\t\tresult := process(i)",
		"\t\tif result > threshold {",
		"\t\t\tfmt.Printf(\"value: %d\\n\", result)",
		"\t\t}",
		"\t}",
		"}",
		"",
	}
	block := strings.Join(lines, "\n")
	reps := (n / len(block)) + 1
	return []byte(strings.Repeat(block, reps)[:n])
}

func BenchmarkEncode100B(b *testing.B) {
	src := makeSource(100)
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for b.Loop() {
		Encode(src)
	}
}

func BenchmarkEncode1KB(b *testing.B) {
	src := makeSource(1024)
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for b.Loop() {
		Encode(src)
	}
}

func BenchmarkEncode4KB(b *testing.B) {
	src := makeSource(4 * 1024)
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for b.Loop() {
		Encode(src)
	}
}

func BenchmarkEncode16KB(b *testing.B) {
	src := makeSource(16 * 1024)
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for b.Loop() {
		Encode(src)
	}
}

func BenchmarkDecode1KB(b *testing.B) {
	src := makeSource(1024)
	enc, _ := Encode(src)
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for b.Loop() {
		Decode(enc)
	}
}

func BenchmarkDecode16KB(b *testing.B) {
	src := makeSource(16 * 1024)
	enc, _ := Encode(src)
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for b.Loop() {
		Decode(enc)
	}
}

func BenchmarkPack7(b *testing.B) {
	src := makeSource(4 * 1024)
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for b.Loop() {
		pack7(src)
	}
}

func BenchmarkUnpack7(b *testing.B) {
	src := makeSource(4 * 1024)
	packed := pack7(src)
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for b.Loop() {
		unpack7(packed)
	}
}
