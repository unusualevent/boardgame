// Package boardgame provides ASCII source code compression and decompression.
//
// Glyphs in the range 0x20–0x73 are stored literally. Repeated sequences
// are placed in a table delimited by 0x00 bytes and referenced with bytes
// 0x01–0x19 (up to 25 entries).
package boardgame

import (
	"errors"
)

const (
	minGlyph = 0x20
	maxGlyph = 0x73
	maxTable = 0x19 // 25 table slots
)

var (
	ErrTooManyEntries  = errors.New("boardgame: table exceeds 25 entries")
	ErrUnterminatedSeq = errors.New("boardgame: unterminated table entry (missing trailing 0x00)")
	ErrBadRef          = errors.New("boardgame: reference to undefined table entry")
	ErrByteOutOfRange  = errors.New("boardgame: byte outside glyph range and not a valid ref")
)

// Encode compresses src by finding repeated substrings, building a table,
// and replacing occurrences with single-byte references.
func Encode(src []byte) ([]byte, error) {
	for _, b := range src {
		if b < minGlyph || b > maxGlyph {
			return nil, ErrByteOutOfRange
		}
	}

	type candidate struct {
		seq   string
		saves int // bytes saved = (count-1)*len - overhead
	}

	used := make(map[string]bool)
	var table []string
	data := string(src)

	for len(table) < maxTable {
		var best candidate
		// try substring lengths 2..len(data)
		for slen := 2; slen <= len(data); slen++ {
			seen := make(map[string]int)
			for i := 0; i <= len(data)-slen; i++ {
				s := data[i : i+slen]
				if used[s] {
					continue
				}
				valid := true
				for j := 0; j < len(s); j++ {
					if s[j] < minGlyph || s[j] > maxGlyph {
						valid = false
						break
					}
				}
				if !valid {
					continue
				}
				seen[s]++
			}
			for s, count := range seen {
				if count < 2 {
					continue
				}
				// count non-overlapping occurrences (left to right)
				nonoverlap := 0
				for i := 0; i <= len(data)-len(s); {
					if data[i:i+len(s)] == s {
						nonoverlap++
						i += len(s)
					} else {
						i++
					}
				}
				if nonoverlap < 2 {
					continue
				}
				// saving: replace nonoverlap occurrences of len(s) with 1 byte each
				// cost: len(s)+2 for the table entry (null + seq + null)
				saves := nonoverlap*len(s) - nonoverlap - (len(s) + 2)
				if saves > best.saves {
					best = candidate{seq: s, saves: saves}
				}
			}
		}
		if best.saves <= 0 {
			break
		}
		table = append(table, best.seq)
		used[best.seq] = true
		// replace occurrences in data with placeholder
		ref := string([]byte{byte(len(table))})
		newData := make([]byte, 0, len(data))
		for i := 0; i < len(data); {
			if i+len(best.seq) <= len(data) && data[i:i+len(best.seq)] == best.seq {
				newData = append(newData, ref[0])
				i += len(best.seq)
			} else {
				newData = append(newData, data[i])
				i++
			}
		}
		data = string(newData)
	}

	var out []byte
	// write table entries
	for _, seq := range table {
		out = append(out, 0x00)
		out = append(out, []byte(seq)...)
		out = append(out, 0x00)
	}
	// write compressed body
	out = append(out, []byte(data)...)
	return out, nil
}

// Decode decompresses a boardgame-encoded byte stream.
func Decode(src []byte) ([]byte, error) {
	// parse table: each entry is delimited by a pair of 0x00 bytes
	var table [][]byte
	i := 0
	for i < len(src) && src[i] == 0x00 {
		i++ // consume opening null
		start := i
		for i < len(src) && src[i] != 0x00 {
			i++
		}
		if i >= len(src) {
			return nil, ErrUnterminatedSeq
		}
		entry := make([]byte, i-start)
		copy(entry, src[start:i])
		table = append(table, entry)
		i++ // consume closing null
		if len(table) > maxTable {
			return nil, ErrTooManyEntries
		}
	}

	// expand body
	var out []byte
	for i < len(src) {
		b := src[i]
		switch {
		case b >= minGlyph && b <= maxGlyph:
			out = append(out, b)
		case b >= 0x01 && b <= maxTable && int(b) <= len(table):
			out = append(out, table[b-1]...)
		default:
			return nil, ErrBadRef
		}
		i++
	}
	return out, nil
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

