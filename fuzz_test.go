package boardgame

import (
	"bytes"
	"testing"
)

// FuzzRoundTrip feeds valid glyph-range input through Encode→Decode
// and checks that the output matches the original.
func FuzzRoundTrip(f *testing.F) {
	f.Add([]byte("hello hello hello"))
	f.Add([]byte("aaa bbb aaa bbb"))
	f.Add([]byte(" "))
	f.Add([]byte("abcabc defdef abcabc"))
	f.Add([]byte("a long phrase here a long phrase here a long phrase here"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// clamp input to valid glyph range
		for i := range data {
			data[i] = minGlyph + data[i]%(maxGlyph-minGlyph+1)
		}

		enc, err := Encode(data)
		if err != nil {
			t.Fatal(err)
		}
		dec, err := Decode(enc)
		if err != nil {
			t.Fatalf("Decode failed: %v (encoded %d bytes from %d)", err, len(enc), len(data))
		}
		if !bytes.Equal(data, dec) {
			t.Errorf("round-trip mismatch:\n  input:   %q\n  decoded: %q", data, dec)
		}
	})
}

// FuzzDecode feeds arbitrary bytes into Decode to check it never panics.
func FuzzDecode(f *testing.F) {
	f.Add([]byte{0x00})
	f.Add([]byte{0x00, 0x41, 0x00, 0x01})
	f.Add([]byte{0x05})
	f.Add([]byte{})
	// a valid packed stream (pack7 of "hello")
	valid := pack7([]byte{0x00, 'h', 'e', 'l', 'l', 'o', 0x00, 0x01})
	f.Add(valid)

	f.Fuzz(func(t *testing.T, data []byte) {
		// must not panic; errors are fine
		Decode(data)
	})
}

// FuzzPack7RoundTrip checks that pack7→unpack7 is lossless for any
// byte sequence that the table layer could produce.
func FuzzPack7RoundTrip(f *testing.F) {
	f.Add([]byte("hello"))
	f.Add([]byte{0x00, 0x41, 0x00, 0x01})
	f.Add([]byte{delByte, 0xFF})

	f.Fuzz(func(t *testing.T, data []byte) {
		// ensure DEL bytes are always paired with a following byte
		var clean []byte
		for i := 0; i < len(data); i++ {
			if data[i] == delByte {
				if i+1 < len(data) {
					clean = append(clean, delByte, data[i+1])
					i++
				}
				// drop unpaired trailing DEL
			} else {
				// keep only 7-bit values
				clean = append(clean, data[i]&0x7E)
			}
		}

		packed := pack7(clean)
		unpacked, err := unpack7(packed)
		if err != nil {
			t.Fatalf("unpack7 failed: %v", err)
		}
		if !bytes.Equal(clean, unpacked) {
			t.Errorf("pack7 round-trip mismatch:\n  input:    %x\n  unpacked: %x", clean, unpacked)
		}
	})
}

// FuzzTableExpandNoPanic feeds arbitrary bytes to tableExpand to ensure
// it never panics regardless of input.
func FuzzTableExpandNoPanic(f *testing.F) {
	f.Add([]byte{0x00, 0x30, 0x00, 0x01})
	f.Add([]byte{0x00, 0x01})
	f.Add([]byte{0x7F, 0x05})
	f.Add([]byte{0x01})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		tableExpand(data)
	})
}
