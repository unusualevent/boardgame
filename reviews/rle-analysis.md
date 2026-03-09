# Run-length encoding analysis

## Motivation

The encoder bottleneck is the suffix array, rebuilt ~25 times per file
at O(n log n) each. RLE as a preprocessing step could shrink the input
before it hits the SA, reducing encode time. The question is whether the
time savings outweigh the compression ratio cost.

## Target characters

**Spaces** — the strongest candidate. Source code indentation produces
runs of 2, 4, 8, 12, 16+ spaces on nearly every line. A typical 4KB Go
file might have 15-20% indentation whitespace.

**Tabs** — same rationale (indentation), though Go uses single tabs per
indent level so runs are less common than in tab-indented languages.

**Newlines** — blank lines produce runs of 2-3 newlines. Short and
infrequent compared to space runs; low impact.

**Other characters** (`=`, `-`, `#` in comments/separators) — too rare
to justify the overhead.

Practical target: **spaces only**, or **spaces and tabs**.

## How RLE could help compression time

- Collapsing `"        "` (8 spaces) into a 2-byte RLE token means the
  SA operates on ~85% of the original input size
- SA construction + LCP scan scale superlinearly, so even a 15% input
  reduction could yield a 25-30% time improvement
- Fewer suffix positions means fewer candidates to evaluate in the LCP
  interval scan
- Helps the worst case most: deeply nested files (Python, YAML, HTML
  templates) have the most whitespace and are the slowest to encode

## Core tension: RLE vs table substitution

Table substitution already discovers patterns that include whitespace:

```
"    return "     →  slot 0x03  (saves 10 bytes per occurrence)
"        "        →  slot 0x05  (saves 7 bytes per occurrence)
"    if "          →  slot 0x07  (saves 5 bytes per occurrence)
```

If RLE pre-encodes `"        "` as `<RLE:space:8>`, the table can no
longer find `"    return "` as a single candidate — it's been split into
`<RLE:space:4>` + `"return "`. The SA would find shorter, less valuable
patterns.

**This could hurt compression ratio even if it helps compression time.**

## Pros

1. **Faster encoding** — smaller input to the SA means fewer positions,
   faster sorts, fewer iterations
2. **Predictable** — space/tab runs are guaranteed present in source code
3. **Simple** — trivial to implement, minimal overhead
4. **Helps the worst case** — deeply nested files have the most
   whitespace and are the slowest to encode

## Cons

1. **Breaks cross-pattern matches** — table substitution loses the
   ability to discover patterns that span whitespace boundaries
   (e.g., `"    return err\n"` as a single entry)
2. **Ratio may decrease** — RLE tokens cost bytes (at least 2 per run),
   and shorter table entries save less per substitution
3. **Interaction with 7-bit packing** — spaces are already 7 bits each;
   RLE tokens would need to be encoded in the same 7-bit stream or
   escape out of it, adding format complexity
4. **Format complexity** — needs a new escape mechanism in the wire
   format, increasing decode complexity
5. **Diminishing returns on small files** — a 200B file might have 20
   bytes of whitespace runs; collapsing them saves microseconds on a
   sub-millisecond encode

## Testing approaches

### 1. Measure whitespace contribution to SA cost

Profile `buildSA` and count what fraction of SA positions come from
whitespace runs. If whitespace positions are < 10% of the SA, RLE
won't help much. Requires no code changes — just instrument:

```go
// in buildSA, count positions that are spaces/tabs
spacePositions := 0
for _, r := range runs {
    for i := r[0]; i < r[1]; i++ {
        if data[i] == ' ' || data[i] == '\t' { spacePositions++ }
    }
}
// compare spacePositions to n
```

This isolates the question of whether whitespace is a meaningful
fraction of the encoder's work. If < 15%, stop here.

### 2. Simulate RLE without changing the wire format

Pre-collapse whitespace runs in the input, run the existing encoder on
the collapsed version, compare encode time. Don't worry about decode —
just measure whether the SA runs faster on smaller input. Isolates the
speed question from the format question.

### 3. Compare ratio with and without RLE

Run the example tool twice — once normal, once with RLE preprocessing —
and diff per-extension ratios. If ratio drops > ~2% overall, the time
savings probably aren't worth it.

### 4. Hybrid: RLE only for runs above a threshold

Only RLE-encode runs of 6+ spaces. Short runs (2-4 spaces) are more
likely part of valuable table patterns like `"  if "`. Long runs (8+
spaces) are pure indentation that rarely appears in useful cross-pattern
matches. Preserves most table substitution value while still shrinking
the SA input.

## Test results

### Approach 1: whitespace fraction of SA positions

Measured across ~/lab monorepo (1341 files, 20KB cap):

| Metric | Value |
|--------|-------|
| Total SA positions | 3,860,112 |
| Whitespace SA positions | 620,604 (16.1%) |
| Space positions | 546,975 (14.2%) |
| Tab positions | 73,629 (1.9%) |
| Space runs (len >= 2) | 40,111 (avg len 6.4) |
| Tab runs (len >= 2) | 18,529 (avg len 2.3) |
| Bytes in whitespace runs | 300,314 |
| SA reduction if collapsed | 4.7% |

The space bucket in the radix sort is **13.7x larger than the average
bucket** (546K positions vs 40K average). This is the largest bucket
by far, meaning the within-bucket comparison sort spends
disproportionate time on space-starting suffixes.

However, the total SA size reduction from collapsing runs is only
4.7%, because the average run length is 5.1 characters — most runs
are short.

SA reduction at different thresholds:

| Threshold | Runs | Bytes in runs | SA reduction |
|-----------|------|---------------|-------------|
| >= 2 | 58,640 | 300,314 | 4.7% |
| >= 4 | 29,402 | 237,201 | 4.6% |
| >= 6 | 18,839 | 193,921 | 4.0% |
| >= 8 | 13,381 | 160,528 | 3.5% |
| >= 12 | 5,114 | 89,202 | 2.0% |

Most of the byte savings come from longer runs; shorter runs are
numerous but contribute little.

### Approach 4: threshold-based RLE simulation

Collapsed whitespace runs to 2 bytes before encoding. Measured both
time and ratio impact. Note: this collapses runs in the input without
a real RLE wire format — the collapsed input is simply smaller.

**20KB cap, threshold >= 4 (1341 files):**

| Metric | Normal | RLE | Change |
|--------|--------|-----|--------|
| Avg ratio | 39.9% | 41.1% | **+1.2% better** |
| Avg time | 30.2ms | 28.4ms | **1.06x faster** |

**20KB cap, threshold >= 6 (1341 files):**

| Metric | Normal | RLE | Change |
|--------|--------|-----|--------|
| Avg ratio | 39.9% | 40.6% | **+0.7% better** |
| Avg time | 31.0ms | 29.5ms | **1.05x faster** |

**Unlimited file size, threshold >= 4 (1404 files):**

| Metric | Normal | RLE | Change |
|--------|--------|-----|--------|
| Avg ratio | 40.8% | 42.0% | **+1.2% better** |
| Avg time | 115.5ms | 99.6ms | **1.16x faster** |

Per-extension highlights (threshold >= 4, 20KB cap):

| Extension | Files | Speedup | Ratio change |
|-----------|-------|---------|-------------|
| .kt | 58 | 1.19x | +3.8% |
| .yaml | 44 | 1.43x | +4.1% |
| .yml | 73 | 1.40x | +1.8% |
| .json | 98 | 1.27x | +2.0% |
| .java | 3 | 1.23x | +3.4% |
| .vue | 96 | 1.14x | +2.3% |
| .html | 55 | 1.16x | +1.2% |
| .go | 372 | 1.05x | +0.5% |

### Surprise: ratio improved

The original concern was that RLE would break cross-pattern matches
and hurt ratio. The opposite happened: collapsing whitespace runs
frees up table slots for more valuable non-whitespace patterns.

Instead of the table spending slots on `"    "` and `"        "`, it
finds higher-value patterns like `"return "`, `"func "`, `"const "`,
etc. The net effect is a ratio improvement.

Files with heavy indentation (Kotlin, YAML, Java) benefit most
because they have the most whitespace to collapse and the most
freed-up table capacity.

### Where it doesn't help

- Files with no whitespace runs (.sum, .d, .lock, .txt): 0% shrink,
  no change in ratio or time
- Very small files (< 200B): noise dominates; no meaningful impact
- Go files: only 1.05x faster because Go uses single tabs (no runs),
  and the space indentation is minimal

## Final implementation

The RLE was implemented with a 2-byte format: {marker, countByte}.

- 0x1F = space RLE marker, 0x1E = tab RLE marker
- Count byte maps run lengths 4–15 to non-literal bytes 0x0B–0x16
- Both marker and count are non-literal, acting as barriers in the suffix array
- Runs > 15 are split into multiple RLE pairs plus remaining literals
- maxDirectRef expanded from 0x19 to 0x1D (4 more single-byte table slots)
- 0x1E and 0x1F are reserved (not usable as table slots)

Pipeline: escapeNonLiteral → rleCompress → tableSubstitute → pack7
Decode: unpack7 → tableExpand → rleExpand

### Failed approaches during implementation

1. **3-byte format with DEL escaping** ({marker, DEL, count}): The count byte (e.g., 0x0A for 10 spaces) was treated as a literal newline by literalRuns, corrupting the SA candidate search. DEL escaping didn't help because pack7 interpreted the DEL as its own escape.

2. **4-byte format** ({marker, hi, lo, marker}): Count split into two non-literal bytes. Correctly avoided literal interference but the 4-byte cost meant break-even only at run length 5. Hurt overall ratio (38.7% vs 39.8% baseline).

3. **RLE after table substitution**: Avoided all interference issues but provided no SA speedup since the table substitution operated on the original (uncollapsed) input. Only compressed leftover spaces, adding 7% overhead with minimal ratio gain.

### Final results (2-byte format, max 15, RLE before table sub)

~/lab monorepo (1335 files, 20KB cap, Intel i7-8700):

| Metric | Pre-RLE baseline | With RLE | Change |
|--------|-----------------|----------|--------|
| Avg ratio | 39.8% | 40.1% | +0.3% better |
| Avg time | 29.6ms | 28.3ms | 1.05x faster |

go test -bench (selected):

| Benchmark | Before | After | Speedup |
|-----------|--------|-------|---------|
| SampleGo | 21.7ms | 19.9ms | 1.09x |
| SampleHTML | 11.1ms | 9.1ms | 1.22x |
| SampleVue | 58ms | 53.3ms | 1.09x |
| SampleRuby | 20.7ms | 19.3ms | 1.07x |

vs standard compressors (1335 files):

| Algorithm | Avg Ratio | Avg Time |
|-----------|----------|----------|
| brotli | 53.5% | 1.9ms |
| zstd | 43.2% | 8.1ms |
| boardgame | 40.1% | 30.3ms |
| gzip | 39.8% | 711us |
| snappy | 37.6% | 45us |
| lz4 | 24.8% | 2.1ms |

Boardgame now beats gzip on ratio (40.1% vs 39.8%), previously tied.

## Conclusion

The 2-byte RLE with nibble-sized max (15) is a net positive:
- Ratio improved (+0.3%) because collapsing whitespace frees table slots for higher-value patterns
- Speed improved (1.05x overall, up to 1.22x on whitespace-heavy files)
- The format is simple: 2 non-literal bytes per RLE pair, no interference with SA or table substitution
- Max run of 15 covers ~95% of whitespace runs (avg run length 6.4)
