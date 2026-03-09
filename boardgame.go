// Package boardgame provides ASCII source code compression and decompression.
//
// Compression has two stages:
//  1. Table substitution — repeated sequences are placed in a table
//     delimited by 0x00 bytes and referenced with bytes 0x01–0x19.
//  2. 7-bit packing — since all ASCII bytes have bit 7 = 0, each byte
//     is stored as 7 bits. The DEL byte (0x7F) escapes a full 8-bit
//     value: the next 8 bits are returned unchanged.
package boardgame

import (
	"errors"
)

const (
	minGlyph = 0x20
	maxGlyph = 0x73
	maxTable = 0x19 // 25 table slots
	delByte  = 0x7F // escape for raw 8-bit values
)

var (
	ErrTooManyEntries  = errors.New("boardgame: table exceeds 25 entries")
	ErrUnterminatedSeq = errors.New("boardgame: unterminated table entry (missing trailing 0x00)")
	ErrBadRef          = errors.New("boardgame: reference to undefined table entry")
	ErrByteOutOfRange  = errors.New("boardgame: byte outside glyph range and not a valid ref")
	ErrTruncated       = errors.New("boardgame: unexpected end of bitstream")
)

// Encode compresses src using table substitution then 7-bit packing.
func Encode(src []byte) ([]byte, error) {
	for _, b := range src {
		if b < minGlyph || b > maxGlyph {
			return nil, ErrByteOutOfRange
		}
	}

	intermediate := tableSubstitute(src)
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

// pack7 packs a byte slice into a 7-bit stream. The first byte of the
// output stores the number of padding bits (0–6) in the final byte.
// Each input byte with bit 7 = 0 is written as 7 bits. Bytes with
// bit 7 = 1 are escaped: DEL (0x7F) followed by the full 8 bits.
func pack7(src []byte) []byte {
	var w bitWriter
	for _, b := range src {
		if b&0x80 != 0 {
			w.writeBits(delByte, 7) // escape
			w.writeBits(uint(b), 8)
		} else {
			w.writeBits(uint(b), 7)
		}
	}
	packed := w.bytes()
	totalBits := w.totalBits
	var padding byte
	if totalBits%8 != 0 {
		padding = byte(8 - totalBits%8)
	}
	return append([]byte{padding}, packed...)
}

// unpack7 unpacks a 7-bit stream back to bytes. The first byte indicates
// how many padding bits trail the data. DEL (0x7F) signals that the next
// 8 bits are a raw byte value.
func unpack7(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}
	padding := int(src[0])
	data := src[1:]
	usableBits := len(data)*8 - padding
	r := bitReader{data: data}
	var out []byte
	for int(r.pos)+7 <= usableBits {
		v, ok := r.readBits(7)
		if !ok {
			break
		}
		if v == delByte {
			if int(r.pos)+8 > usableBits {
				return nil, ErrTruncated
			}
			raw, ok := r.readBits(8)
			if !ok {
				return nil, ErrTruncated
			}
			out = append(out, byte(raw))
		} else {
			out = append(out, byte(v))
		}
	}
	return out, nil
}

// tableSubstitute finds repeated substrings and replaces them with
// single-byte references (0x01–0x19), returning the table + body.
func tableSubstitute(src []byte) []byte {
	type candidate struct {
		seq   string
		saves int
	}

	used := make(map[string]bool)
	var table []string
	data := string(src)

	for len(table) < maxTable {
		var best candidate
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
	for _, seq := range table {
		out = append(out, 0x00)
		out = append(out, []byte(seq)...)
		out = append(out, 0x00)
	}
	out = append(out, []byte(data)...)
	return out
}

// tableExpand parses the null-delimited table and expands references.
func tableExpand(src []byte) ([]byte, error) {
	var table [][]byte
	i := 0
	for i < len(src) && src[i] == 0x00 {
		i++
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
		i++
		if len(table) > maxTable {
			return nil, ErrTooManyEntries
		}
	}

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

// bitWriter accumulates bits into a byte slice, MSB first.
type bitWriter struct {
	buf       []byte
	n         uint // bits written into current byte
	totalBits int
}

func (w *bitWriter) writeBits(val uint, nbits int) {
	w.totalBits += nbits
	for i := nbits - 1; i >= 0; i-- {
		if w.n == 0 {
			w.buf = append(w.buf, 0)
		}
		if val&(1<<uint(i)) != 0 {
			w.buf[len(w.buf)-1] |= 1 << (7 - w.n)
		}
		w.n++
		if w.n == 8 {
			w.n = 0
		}
	}
}

func (w *bitWriter) bytes() []byte {
	return w.buf
}

// bitReader reads bits from a byte slice, MSB first.
type bitReader struct {
	data []byte
	pos  uint // bit position
}

func (r *bitReader) readBits(n int) (uint, bool) {
	if r.remaining() < n {
		return 0, false
	}
	var val uint
	for i := 0; i < n; i++ {
		byteIdx := r.pos / 8
		bitIdx := 7 - (r.pos % 8)
		if r.data[byteIdx]&(1<<bitIdx) != 0 {
			val |= 1 << uint(n-1-i)
		}
		r.pos++
	}
	return val, true
}

func (r *bitReader) remaining() int {
	total := len(r.data) * 8
	return total - int(r.pos)
}
