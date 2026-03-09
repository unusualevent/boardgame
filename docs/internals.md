# Internals

This document describes the internal architecture of the boardgame codec.

## Compression pipeline

```
Source bytes ──► tableSubstitute ──► intermediate stream ──► pack7 ──► compressed output
```

## Decompression pipeline

```
Compressed input ──► unpack7 ──► intermediate stream ──► tableExpand ──► source bytes
```

## Table substitution (`tableSubstitute`)

A greedy algorithm that iteratively finds the best repeated substring to
replace:

1. Find the lowest free slot (1–255).
2. Find the best candidate substring using suffix-array-based search.
3. Score each candidate:
   `saves = occurrences * len(seq) - occurrences * refCost(slot) - (len(seq) + 2)`
4. Pick the candidate with the highest positive savings.
5. Replace all non-overlapping occurrences with the reference byte
   (or null-DEL-byte for extended slots).
6. Repeat until no candidate yields positive savings or all slots are
   full.

The output is the table definitions followed by the substituted body.
Slots 0x01–0x19 use a 1-byte reference; slots 0x1A–0xFF use a 3-byte
`{0x00}{0x7F}{slot}` reference, so the savings threshold is higher for
extended slots.

### Candidate search via suffix array

The candidate search (step 2) uses a suffix array built over only the
literal-byte positions in the intermediate stream. Non-literal bytes
(existing references from prior rounds) act as hard boundaries that
candidates cannot span.

Helper functions:

- **`literalRuns`** — O(n) scan to identify contiguous runs of literal
  bytes (printable ASCII, tab, newline). Only runs of length >= 2 can
  contain candidates.
- **`buildSA`** — builds a suffix array over literal-run positions,
  sorted lexicographically with run boundaries acting as terminators.
  O(m log m) via `sort.Slice`.
- **`buildLCP`** — computes the longest common prefix array for adjacent
  suffix array entries, clamped to run boundaries. O(m) per pair.
- **`nonOverlapCount`** — greedy left-to-right non-overlapping count
  from sorted positions. O(k) where k = group size.
- **`findBestCandidate`** — for each candidate length L from 2 to
  maxLCP, walks the suffix array to identify groups of suffixes sharing
  a prefix of length >= L, scores each group, and tracks the global
  best. Overall O(maxLCP * m) per greedy iteration.

This replaces the original O(n^3) brute-force search with an
O(m log m + maxLCP * m) search per iteration, where m is the number of
literal positions (shrinks each round) and maxLCP is the longest repeated
literal-only prefix (typically small for source code).

## Table expansion (`tableExpand`)

Processes the intermediate stream byte by byte. A `0x00` byte triggers
one of three actions depending on the next byte:

- `0x7F` → extended reference: read one more byte as the slot ID
- `0x01–0x19` (occupied) → free that slot
- anything else → start of a new table entry, scan to closing `0x00`

Direct references (`0x01–0x19`) and literals (`0x20–0x7E`) are handled
inline. Standalone `0x7F` passes the next byte through as a literal.

## 7-bit packing (`pack7` / `unpack7`)

### bitWriter

Accumulates bits MSB-first into a byte slice. Tracks total bits written
so the padding count can be computed.

### bitReader

Reads bits MSB-first from a byte slice. Tracks a bit position cursor.

### DEL escape in the bitstream

When `pack7` encounters `0x7F` in the input, it writes 7 bits (the DEL
value) followed by the next byte as 8 bits. This preserves bytes with
bit 7 set that the table layer produces via DEL escapes. `unpack7`
mirrors this: on reading a 7-bit DEL, it reads the next 8 bits and emits
both the DEL marker and the raw byte.

### Padding

The packed output is prefixed with a single byte indicating how many
padding bits (0–6) were appended to fill the last byte. The unpacker
uses this to know when to stop reading.

## Slot management

`lowestFreeSlot` scans slots 1–255 and returns the first unoccupied one
(or 0 if all are full). This ensures direct slots (cheaper references)
are always consumed before extended slots.
