// Package boardgame provides ASCII source code compression and decompression.
//
// Compression has two stages:
//  1. Table substitution —repeated sequences are placed in a table
//     delimited by 0x00 bytes and referenced with bytes 0x01–0x19
//     (up to 25 direct slots). An unpaired null followed by a ref byte
//     (0x01–0x19) frees that slot. New entries always claim the lowest
//     free slot. The sequence {null}{DEL}{byte} extends references to
//     a full 8-bit range (slots 0x1A–0xFF), allowing up to 255 entries.
//  2. 7-bit packing —since all ASCII bytes have bit 7 = 0, each byte
//     is stored as 7 bits. The DEL byte (0x7F) escapes a full 8-bit
//     value: the next 8 bits are returned unchanged.
package boardgame

import (
	"errors"
	"sort"
)

const (
	minGlyph       = 0x20
	maxGlyph       = 0x7E
	maxDirectRef   = 0x19 // slots 0x01–0x19: single-byte reference
	maxExtRef      = 0xFF // slots 0x1A–0xFF: null-DEL-byte reference
	delByte        = 0x7F // escape for raw 8-bit values
	tab            = 0x09
	newline        = 0x0A
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

// pack7 packs a byte slice into a 7-bit stream. The first byte of the
// output stores the number of padding bits (0–6) in the final byte.
// Each input byte with bit 7 = 0 is written as 7 bits. DEL (0x7F)
// in the input signals that the next byte is written as 8 raw bits.
func pack7(src []byte) []byte {
	var w bitWriter
	for i := 0; i < len(src); i++ {
		b := src[i]
		if b == delByte {
			w.writeBits(delByte, 7)
			i++
			if i < len(src) {
				w.writeBits(uint(src[i]), 8)
			}
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
// 8 bits are a raw byte value; both the DEL marker and the raw byte are
// emitted so the table layer can see them.
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
			out = append(out, delByte, byte(raw))
		} else {
			out = append(out, byte(v))
		}
	}
	return out, nil
}

// lowestFreeSlot returns the lowest unused slot number (1–0xFF),
// or 0 if all 255 slots are occupied. Slots reserved for literal
// bytes (tab, newline) are skipped.
func lowestFreeSlot(table map[byte][]byte) byte {
	for s := byte(1); s != 0; s++ { // 1..255, wraps to 0
		if isReserved(s) {
			continue
		}
		if _, ok := table[s]; !ok {
			return s
		}
	}
	return 0
}

// literalRuns returns [start, end) pairs of maximal contiguous runs of
// isLiteral bytes in data. Only runs of length >= 2 are included.
func literalRuns(data string) [][2]int {
	var runs [][2]int
	i := 0
	for i < len(data) {
		if !isLiteral(data[i]) {
			i++
			continue
		}
		start := i
		for i < len(data) && isLiteral(data[i]) {
			i++
		}
		if i-start >= 2 {
			runs = append(runs, [2]int{start, i})
		}
	}
	return runs
}

// buildSA builds a suffix array over only the literal-run positions in data.
// It also returns runEnd where runEnd[i] is the end of i's literal run
// (or 0 if i is not in a run). Suffixes are compared only up to their run
// boundary, so substrings never span across non-literal bytes.
func buildSA(data string, runs [][2]int) ([]int, []int) {
	runEnd := make([]int, len(data))
	var sa []int
	for _, r := range runs {
		for i := r[0]; i < r[1]; i++ {
			runEnd[i] = r[1]
			sa = append(sa, i)
		}
	}
	sort.Slice(sa, func(a, b int) bool {
		ai, bi := sa[a], sa[b]
		ae, be := runEnd[ai], runEnd[bi]
		la, lb := ae-ai, be-bi
		ml := la
		if lb < ml {
			ml = lb
		}
		for k := 0; k < ml; k++ {
			if data[ai+k] != data[bi+k] {
				return data[ai+k] < data[bi+k]
			}
		}
		return la < lb
	})
	return sa, runEnd
}

// buildLCP computes the LCP array for adjacent SA entries, clamped to
// literal-run boundaries via runEnd.
func buildLCP(data string, sa []int, runEnd []int) []int {
	n := len(sa)
	if n == 0 {
		return nil
	}
	lcp := make([]int, n)
	for i := 1; i < n; i++ {
		a, b := sa[i-1], sa[i]
		maxA, maxB := runEnd[a]-a, runEnd[b]-b
		ml := maxA
		if maxB < ml {
			ml = maxB
		}
		h := 0
		for h < ml && data[a+h] == data[b+h] {
			h++
		}
		lcp[i] = h
	}
	return lcp
}

// nonOverlapCount returns the greedy left-to-right non-overlapping count
// of a substring of the given length at the given sorted positions.
func nonOverlapCount(positions []int, length int) int {
	count := 0
	barrier := -1
	for _, p := range positions {
		if p >= barrier {
			count++
			barrier = p + length
		}
	}
	return count
}

// findBestCandidate searches for the repeated literal-only substring with
// the highest savings score. It uses a suffix array and LCP array to
// enumerate candidates efficiently.
func findBestCandidate(data string, rc int, used map[string]bool) (string, int) {
	runs := literalRuns(data)
	if len(runs) == 0 {
		return "", 0
	}
	sa, runEnd := buildSA(data, runs)
	if len(sa) < 2 {
		return "", 0
	}
	lcp := buildLCP(data, sa, runEnd)

	maxLCP := 0
	for _, v := range lcp {
		if v > maxLCP {
			maxLCP = v
		}
	}
	if maxLCP < 2 {
		return "", 0
	}

	var bestSeq string
	var bestSaves int

	// For each candidate length L, walk the SA to find groups of suffixes
	// sharing a prefix of length >= L.
	for L := 2; L <= maxLCP; L++ {
		i := 0
		for i < len(sa) {
			// Find the end of this group: consecutive SA entries with lcp >= L.
			j := i + 1
			for j < len(sa) && lcp[j] >= L {
				j++
			}
			groupSize := j - i
			if groupSize >= 2 {
				pos := sa[i]
				seq := data[pos : pos+L]
				if !used[seq] {
					// Collect and sort positions for non-overlap counting.
					sorted := make([]int, groupSize)
					copy(sorted, sa[i:j])
					sort.Ints(sorted)
					nonoverlap := nonOverlapCount(sorted, L)
					if nonoverlap >= 2 {
						saves := nonoverlap*L - nonoverlap*rc - (L + 2)
						if saves > bestSaves {
							bestSaves = saves
							bestSeq = seq
						}
					}
				}
			}
			i = j
		}
	}
	return bestSeq, bestSaves
}

// tableSubstitute finds repeated substrings and replaces them with
// references, always assigning the lowest free slot. Direct slots
// (0x01–0x19) use a single-byte ref; extended slots (0x1A–0xFF)
// use the 3-byte sequence {null}{DEL}{slot}.
func tableSubstitute(src []byte) []byte {
	type candidate struct {
		seq   string
		saves int
	}

	used := make(map[string]bool)
	table := make(map[byte]string)
	data := string(src)

	for {
		slot := lowestFreeSlot(byteMapToCheck(table))
		if slot == 0 {
			break
		}
		rc := refCost(slot)

		bestSeq, bestSaves := findBestCandidate(data, rc, used)
		best := candidate{seq: bestSeq, saves: bestSaves}
		if best.saves <= 0 {
			break
		}
		table[slot] = best.seq
		used[best.seq] = true

		// replace occurrences in data
		newData := make([]byte, 0, len(data))
		for i := 0; i < len(data); {
			if i+len(best.seq) <= len(data) && data[i:i+len(best.seq)] == best.seq {
				if slot <= maxDirectRef {
					newData = append(newData, slot)
				} else {
					newData = append(newData, 0x00, delByte, slot)
				}
				i += len(best.seq)
			} else {
				newData = append(newData, data[i])
				i++
			}
		}
		data = string(newData)
	}

	// emit table entries in slot order
	var out []byte
	for s := byte(1); s != 0; s++ {
		seq, ok := table[s]
		if !ok {
			continue
		}
		out = append(out, 0x00)
		out = append(out, []byte(seq)...)
		out = append(out, 0x00)
	}
	out = append(out, []byte(data)...)
	return out
}

// byteMapToCheck converts a map[byte]string to map[byte][]byte for
// lowestFreeSlot compatibility.
func byteMapToCheck(m map[byte]string) map[byte][]byte {
	r := make(map[byte][]byte, len(m))
	for k, v := range m {
		r[k] = []byte(v)
	}
	return r
}

// tableExpand processes the intermediate stream: defines table entries
// from null-delimited sequences, frees slots on unpaired null + ref,
// handles extended references via null-DEL-byte, and expands references.
func tableExpand(src []byte) ([]byte, error) {
	table := make(map[byte][]byte)
	var out []byte
	i := 0
	for i < len(src) {
		b := src[i]
		switch {
		case b == 0x00:
			i++
			if i >= len(src) {
				return nil, ErrUnterminatedSeq
			}
			next := src[i]

			// null-DEL-byte: extended 8-bit table reference
			if next == delByte {
				i++
				if i >= len(src) {
					return nil, ErrTruncated
				}
				ref := src[i]
				entry, ok := table[ref]
				if !ok {
					return nil, ErrBadRef
				}
				out = append(out, entry...)
				i++
				continue
			}

			// unpaired null + direct ref byte: free that slot
			if next >= 0x01 && next <= maxDirectRef {
				if _, occupied := table[next]; occupied {
					delete(table, next)
					i++
					continue
				}
			}

			// paired null: define a new table entry
			start := i
			for i < len(src) && src[i] != 0x00 {
				i++
			}
			if i >= len(src) {
				return nil, ErrUnterminatedSeq
			}
			entry := make([]byte, i-start)
			copy(entry, src[start:i])
			slot := lowestFreeSlot(table)
			if slot == 0 {
				return nil, ErrTooManyEntries
			}
			table[slot] = entry
			i++ // consume closing null

		case b == delByte:
			// DEL escape: next byte is a literal 8-bit value
			i++
			if i >= len(src) {
				return nil, ErrTruncated
			}
			out = append(out, src[i])
			i++

		case b == tab || b == newline:
			out = append(out, b)
			i++

		case b >= 0x01 && b <= maxDirectRef:
			entry, ok := table[b]
			if !ok {
				return nil, ErrBadRef
			}
			out = append(out, entry...)
			i++

		case b >= minGlyph && b <= maxGlyph:
			out = append(out, b)
			i++

		default:
			return nil, ErrBadRef
		}
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
