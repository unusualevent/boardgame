// Package boardgame provides ASCII source code compression and decompression.
//
// Compression has two stages:
//  1. Table substitution — repeated sequences are placed in a table
//     delimited by 0x00 bytes and referenced with bytes 0x01–0x19
//     (up to 25 direct slots). An unpaired null followed by a ref byte
//     (0x01–0x19) frees that slot. New entries always claim the lowest
//     free slot. The sequence {null}{DEL}{byte} extends references to
//     a full 8-bit range (slots 0x1A–0xFF), allowing up to 255 entries.
//  2. 7-bit packing — since all ASCII bytes have bit 7 = 0, each byte
//     is stored as 7 bits. The DEL byte (0x7F) escapes a full 8-bit
//     value: the next 8 bits are returned unchanged.
package boardgame

import "errors"

const (
	minGlyph     = 0x20
	maxGlyph     = 0x7E
	maxDirectRef = 0x19 // slots 0x01–0x19: single-byte reference
	maxExtRef    = 0xFF // slots 0x1A–0xFF: null-DEL-byte reference
	delByte      = 0x7F // escape for raw 8-bit values
	tab          = 0x09
	newline      = 0x0A
)

// isLiteral reports whether b is a valid literal byte in the intermediate
// stream (printable glyphs, tab, or newline).
func isLiteral(b byte) bool {
	return (b >= minGlyph && b <= maxGlyph) || b == tab || b == newline
}

// isReserved reports whether b is reserved and must not be used as a table
// slot ID (tab and newline are literal bytes, not references).
func isReserved(b byte) bool {
	return b == tab || b == newline
}

var (
	ErrTooManyEntries  = errors.New("boardgame: table exceeds 255 entries")
	ErrUnterminatedSeq = errors.New("boardgame: unterminated table entry (missing trailing 0x00)")
	ErrBadRef          = errors.New("boardgame: reference to undefined table entry")
	ErrByteOutOfRange  = errors.New("boardgame: byte outside glyph range and not a valid ref")
	ErrTruncated       = errors.New("boardgame: unexpected end of bitstream")
	ErrNoFreeSlot      = errors.New("boardgame: no free table slot available")
)

// refCost returns the byte cost of referencing a given slot in the
// intermediate stream: 1 byte for direct slots, 3 for extended.
func refCost(slot byte) int {
	if slot >= 0x01 && slot <= maxDirectRef {
		return 1
	}
	return 3 // 0x00 + 0x7F + slot
}

// escapeNonLiteral prepends a DEL escape byte before each non-literal
// byte in src, allowing UTF-8 and other non-ASCII content to pass
// through the compression pipeline. Non-literal bytes act as barriers
// in the candidate search but round-trip correctly.
func escapeNonLiteral(src []byte) []byte {
	n := 0
	for _, b := range src {
		if !isLiteral(b) {
			n++
		}
	}
	if n == 0 {
		return src
	}
	out := make([]byte, 0, len(src)+n)
	for _, b := range src {
		if !isLiteral(b) {
			out = append(out, delByte, b)
		} else {
			out = append(out, b)
		}
	}
	return out
}

// Encode compresses src using table substitution then 7-bit packing.
// Non-ASCII bytes (UTF-8, etc.) are DEL-escaped and pass through
// transparently; only ASCII portions participate in compression.
func Encode(src []byte) ([]byte, error) {
	escaped := escapeNonLiteral(src)
	intermediate := tableSubstitute(escaped)
	return pack7(intermediate), nil
}

// Decode decompresses a boardgame-encoded byte stream.
func Decode(src []byte) ([]byte, error) {
	unpacked, err := unpack7(src)
	if err != nil {
		return nil, err
	}
	return tableExpand(unpacked)
}

// Stats returns compression statistics for the given input.
func Stats(original, compressed []byte) (origLen, compLen int, ratio float64) {
	origLen = len(original)
	compLen = len(compressed)
	if origLen > 0 {
		ratio = 1.0 - float64(compLen)/float64(origLen)
	}
	return
}
