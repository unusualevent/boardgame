// Package boardgame provides ASCII source code compression and decompression.
//
// Compression has three stages:
//  1. RLE — runs of 4–15 spaces or tabs are collapsed to 2 bytes:
//     {0x1F, count} for spaces, {0x1E, count} for tabs. Both the
//     marker and count byte are non-literal, acting as barriers in
//     the suffix array candidate search.
//  2. Table substitution — repeated sequences are placed in a table
//     delimited by 0x00 bytes and referenced with bytes 0x01–0x1D
//     (up to 27 direct slots). An unpaired null followed by a ref byte
//     (0x01–0x1D) frees that slot. The sequence {null}{DEL}{byte}
//     extends references to a full 8-bit range (slots 0x1E–0xFF),
//     allowing up to 251 entries.
//  3. 7-bit packing — since all ASCII bytes have bit 7 = 0, each byte
//     is stored as 7 bits. The DEL byte (0x7F) escapes a full 8-bit
//     value: the next 8 bits are returned unchanged.
package boardgame

import "errors"

const (
	minGlyph     = 0x20
	maxGlyph     = 0x7E
	maxDirectRef = 0x1D // slots 0x01–0x1D: single-byte reference
	maxExtRef    = 0xFF // slots 0x1E–0xFF: null-DEL-byte reference
	delByte      = 0x7F // escape for raw 8-bit values
	tab          = 0x09
	newline      = 0x0A
	rleTab       = 0x1E // RLE indicator for tab runs
	rleSpace     = 0x1F // RLE indicator for space runs
	rleMinRun    = 4    // minimum run length to RLE-encode
	rleMaxRun    = 15   // maximum run length per RLE pair (nibble)
	rleCountBase = 0x0B // count byte = run_length - rleMinRun + rleCountBase
)

// isLiteral reports whether b is a valid literal byte in the intermediate
// stream (printable glyphs, tab, or newline).
func isLiteral(b byte) bool {
	return (b >= minGlyph && b <= maxGlyph) || b == tab || b == newline
}

// isReserved reports whether b is reserved and must not be used as a table
// slot ID (tab, newline, and RLE indicators are not table slots).
func isReserved(b byte) bool {
	return b == tab || b == newline || b == rleTab || b == rleSpace
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

// rleCompress replaces runs of 4–15 spaces or tabs with 2-byte RLE
// sequences: {marker, countByte}. The count byte maps run lengths
// 4–15 to non-literal bytes 0x0B–0x16, ensuring neither the marker
// nor the count participates in literal runs or SA candidate search.
// Runs longer than 15 are split into multiple RLE pairs plus
// remaining literal characters.
func rleCompress(src []byte) []byte {
	out := make([]byte, 0, len(src))
	i := 0
	for i < len(src) {
		if src[i] == ' ' || src[i] == tab {
			ch := src[i]
			start := i
			for i < len(src) && src[i] == ch {
				i++
			}
			rl := i - start
			if rl >= rleMinRun {
				marker := byte(rleSpace)
				if ch == tab {
					marker = rleTab
				}
				for rl >= rleMinRun {
					n := rl
					if n > rleMaxRun {
						n = rleMaxRun
					}
					out = append(out, marker, byte(n-rleMinRun)+rleCountBase)
					rl -= n
				}
				// Remaining < rleMinRun chars emitted as literals.
				for range rl {
					out = append(out, ch)
				}
			} else {
				out = append(out, src[start:i]...)
			}
		} else {
			out = append(out, src[i])
			i++
		}
	}
	return out
}

// rleExpand restores RLE-compressed runs of spaces and tabs.
func rleExpand(src []byte) ([]byte, error) {
	out := make([]byte, 0, len(src))
	i := 0
	for i < len(src) {
		if src[i] == rleSpace || src[i] == rleTab {
			ch := byte(' ')
			if src[i] == rleTab {
				ch = tab
			}
			i++
			if i >= len(src) {
				return nil, ErrTruncated
			}
			count := int(src[i]-rleCountBase) + rleMinRun
			i++
			for range count {
				out = append(out, ch)
			}
		} else {
			out = append(out, src[i])
			i++
		}
	}
	return out, nil
}

// Encode compresses src using RLE, table substitution, then 7-bit packing.
// Non-ASCII bytes (UTF-8, etc.) are DEL-escaped and pass through
// transparently; only ASCII portions participate in compression.
func Encode(src []byte) ([]byte, error) {
	escaped := escapeNonLiteral(src)
	rled := rleCompress(escaped)
	intermediate := tableSubstitute(rled)
	return pack7(intermediate), nil
}

// Decode decompresses a boardgame-encoded byte stream.
func Decode(src []byte) ([]byte, error) {
	unpacked, err := unpack7(src)
	if err != nil {
		return nil, err
	}
	expanded, err := tableExpand(unpacked)
	if err != nil {
		return nil, err
	}
	return rleExpand(expanded)
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
