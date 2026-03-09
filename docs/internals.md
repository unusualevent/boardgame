# Internals

This document describes the internal architecture of the boardgame codec.

## Compression pipeline

```
Source bytes ──► escapeNonLiteral ──► rleCompress ──► tableSubstitute ──► intermediate stream ──► pack7 ──► compressed output
```

### Non-ASCII escaping (`escapeNonLiteral`)

Before table substitution, any byte that is not a literal (not printable
ASCII, tab, or newline) is prefixed with a DEL escape byte (`0x7F`).
This includes UTF-8 continuation bytes, null bytes, and DEL itself.
The escaped bytes act as barriers in the candidate search — they will
not participate in dictionary compression — but round-trip correctly
through the rest of the pipeline since `pack7`, `unpack7`, and
`tableExpand` all handle DEL escapes natively.

### RLE preprocessing (`rleCompress`)

Runs of 4–15 consecutive spaces or tabs are collapsed to 2-byte tokens:
`{marker, count_byte}`. The space marker is `0x1F`, the tab marker is
`0x1E`. The count byte maps run lengths 4–15 to non-literal bytes
`0x0B–0x16` via `count_byte = run_length - 4 + 0x0B`. Both bytes are
non-literal, so they act as barriers in `literalRuns` and the suffix
array candidate search. Runs longer than 15 are split into multiple
RLE pairs plus remaining literal characters.

This shrinks the input before it reaches the suffix array, reducing
encode time (1.05x overall, up to 1.22x on whitespace-heavy files)
and freeing table slots for higher-value patterns (+0.3% ratio).

## Decompression pipeline

```
Compressed input ──► unpack7 ──► intermediate stream ──► tableExpand ──► rleExpand ──► source bytes
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
Slots 0x01–0x1D use a 1-byte reference; slots 0x1E–0xFF use a 3-byte
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
- **`findBestCandidate`** — uses a single-pass stack-based LCP interval
  decomposition to enumerate all candidate substrings in O(m) time.
  Each time an LCP value drops, intervals with longer shared prefixes
  are closed and evaluated as candidates.

This replaces the original O(n^3) brute-force search with an
O(m log m) search per iteration (dominated by the SA sort), where m
is the number of literal positions (shrinks each round).

## Table expansion (`tableExpand`)

Processes the intermediate stream byte by byte. A `0x00` byte triggers
one of three actions depending on the next byte:

- `0x7F` → extended reference: read one more byte as the slot ID
- `0x01–0x1D` (occupied) → free that slot
- anything else → start of a new table entry, scan to closing `0x00`

Direct references (`0x01–0x1D`) and literals (`0x20–0x7E`) are handled
inline. RLE markers (`0x1E`, `0x1F`) are passed through with their count
byte for `rleExpand` to restore. Standalone `0x7F` passes the next byte
through as a literal.

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

`lowestFreeSlot` scans slots 1–255, skipping reserved bytes (`0x09`,
`0x0A`, `0x1E`, `0x1F`), and returns the first unoccupied one (or 0 if
all are full). This ensures direct slots (cheaper 1-byte references)
are always consumed before extended slots (3-byte references). With 4
reserved bytes, a maximum of 251 slots are available.
