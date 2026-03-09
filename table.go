package boardgame

import "sort"

// lowestFreeSlot returns the lowest unused slot number (1–0xFF),
// or 0 if all 255 slots are occupied. Slots reserved for literal
// bytes (tab, newline) are skipped.
func lowestFreeSlot(table map[byte]string) byte {
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

// lcpInterval tracks an open LCP interval on the stack during the
// single-pass candidate scan.
type lcpInterval struct {
	length int // shared prefix length for this interval
	start  int // SA index where this interval begins
}

// findBestCandidate searches for the repeated literal-only substring with
// the highest savings score. It uses a suffix array and LCP array to
// enumerate candidates in a single O(n) pass via a stack-based LCP
// interval decomposition, rather than iterating over all lengths.
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

	var bestSeq string
	var bestSaves int
	var scratch []int // reused across evaluations

	// evaluate scores a candidate interval [start, end) at the given length.
	evaluate := func(length, start, end int) {
		if length < 2 {
			return
		}
		groupSize := end - start
		if groupSize < 2 {
			return
		}
		pos := sa[start]
		seq := data[pos : pos+length]
		if used[seq] {
			return
		}
		scratch = append(scratch[:0], sa[start:end]...)
		sort.Ints(scratch)
		nonoverlap := nonOverlapCount(scratch, length)
		if nonoverlap < 2 {
			return
		}
		saves := nonoverlap*length - nonoverlap*rc - (length + 2)
		if saves > bestSaves {
			bestSaves = saves
			bestSeq = seq
		}
	}

	// Single-pass stack-based LCP interval scan. Each time an LCP value
	// drops, we close all intervals with length > current LCP, evaluating
	// them as candidates. This visits each SA entry exactly once.
	var stack []lcpInterval
	for i := 1; i < len(sa); i++ {
		start := i - 1
		for len(stack) > 0 && stack[len(stack)-1].length > lcp[i] {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			evaluate(top.length, top.start, i)
			start = top.start
		}
		if len(stack) == 0 || stack[len(stack)-1].length < lcp[i] {
			stack = append(stack, lcpInterval{length: lcp[i], start: start})
		}
	}
	// Flush remaining intervals.
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		evaluate(top.length, top.start, len(sa))
	}

	return bestSeq, bestSaves
}

// tableSubstitute finds repeated substrings and replaces them with
// references, always assigning the lowest free slot. Direct slots
// (0x01–0x19) use a single-byte ref; extended slots (0x1A–0xFF)
// use the 3-byte sequence {null}{DEL}{slot}.
func tableSubstitute(src []byte) []byte {
	used := make(map[string]bool)
	table := make(map[byte]string)
	data := string(src)

	for {
		slot := lowestFreeSlot(table)
		if slot == 0 {
			break
		}
		rc := refCost(slot)

		bestSeq, bestSaves := findBestCandidate(data, rc, used)
		if bestSaves <= 0 {
			break
		}
		table[slot] = bestSeq
		used[bestSeq] = true

		// replace occurrences in data
		newData := make([]byte, 0, len(data))
		for i := 0; i < len(data); {
			if i+len(bestSeq) <= len(data) && data[i:i+len(bestSeq)] == bestSeq {
				if slot <= maxDirectRef {
					newData = append(newData, slot)
				} else {
					newData = append(newData, 0x00, delByte, slot)
				}
				i += len(bestSeq)
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

// tableExpand processes the intermediate stream: defines table entries
// from null-delimited sequences, frees slots on unpaired null + ref,
// handles extended references via null-DEL-byte, and expands references.
func tableExpand(src []byte) ([]byte, error) {
	table := make(map[byte]string)
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
			entry := string(src[start:i])
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
