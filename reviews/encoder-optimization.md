# Encoder optimization plan

## Profile summary (4KB encode, Intel i7-8700 @ 3.20GHz)

| Where | CPU | Memory | What |
|---|---|---|---|
| `findBestCandidate` | **85.5%** | **628 MB (91%)** | Dominates everything |
| `buildSA` (suffix array sort) | 21.4% | 40 MB | Rebuilt from scratch every iteration |
| `sort.Ints` (position sorting) | 23.5% | — | Sorting candidate positions per group |
| `make([]int, groupSize)` | — | 140ms | Allocating position slices per group |
| `buildLCP` | 5.5% | 9 MB | LCP recomputed every iteration |
| `pack7`/`unpack7` | <1% | <1% | Already fast, not worth optimizing |

Core problem: `findBestCandidate` is called once per greedy iteration
(up to ~25 useful slots), and each call rebuilds the entire suffix array
from scratch. For a 4KB input that's ~25 full SA constructions + LCP
computations + exhaustive candidate walks.

## Baseline (reverted, ~/lab monorepo)

- 1331 files, 39.8% avg compression, **63.4ms avg encode time**
- `.java` (16.5KB): 401ms, `.vue` (5.3KB): 149ms, `.go` (4.8KB): 131ms

## Implementation order

### 1. Eliminate per-group position slice allocation
The `make([]int, groupSize)` + `sort.Ints` inside the inner loop accounts
for 23.5% of CPU and most of the 553K allocs. Use a single pre-allocated
scratch buffer grown with `append`, reused across groups within each
`findBestCandidate` call.

### 2. Single-pass LCP interval scan (replace O(maxLCP * n) candidate scan)
The inner loop `for L := 2; L <= maxLCP; L++` walks the entire SA for
every candidate length. A stack-based LCP interval tree approach finds the
best candidate in a single O(n) sweep instead of O(maxLCP * n).

### 3. Stop rebuilding SA every iteration
Build the suffix array once, maintain a candidate structure. After each
substitution, invalidate affected positions rather than rebuilding from
scratch. Attacks the 21% `buildSA` cost multiplied by ~25 iterations.

### 4. Radix/counting sort for positions
`sort.Ints` uses pdqsort (comparison-based). Positions are bounded integers
(0..len(data)), so counting sort or radix sort is O(n) instead of
O(n log n).

## Results

### After optimization 1: scratch buffer (commit 3681f10)

Allocs dropped massively. Time modestly faster on controlled benchmarks.

**go test -bench (count=3 medians):**

| Benchmark | Before (time) | After (time) | Before (allocs) | After (allocs) |
|---|---|---|---|---|
| Encode4KB | 105ms | **86ms** | 553,196 | **103** |
| RealLibrary | 22.2ms | **21.9ms** | 75,820 | **1,195** |
| RealTests | 189ms | **177ms** | 847,551 | **3,464** |
| SampleGo | 51ms | **46ms** | 266,908 | **1,707** |
| SampleHTML | 26ms | **22ms** | 125,121 | **1,110** |
| SampleJS | 47ms | **47ms** | 167,177 | **2,086** |
| SampleJSON | 9.6ms | **8.9ms** | 45,653 | **792** |
| SampleVue | 112ms | **127ms** (noisy) | 380,339 | **2,964** |
| SampleRuby | 36ms | **55ms** (noisy) | 125,302 | **1,937** |

### After optimization 2: LCP interval scan (pending commit)

Replaced O(maxLCP * n) nested loop with single-pass O(n) stack-based
LCP interval decomposition. Huge speedup across all benchmarks.

**go test -bench (count=3 medians):**

| Benchmark | Before (scratch only) | After (LCP scan) | Speedup |
|---|---|---|---|
| Encode4KB | 86ms | **23.5ms** | **3.7x** |
| RealLibrary | 21.9ms | **16.4ms** | **1.34x** |
| RealTests | 177ms | **107ms** | **1.65x** |
| SampleGo | 46ms | **29ms** | **1.59x** |
| SampleHTML | 22ms | **14.3ms** | **1.54x** |
| SampleJS | 47ms | **36.5ms** | **1.29x** |
| SampleJSON | 8.9ms | **5.9ms** | **1.51x** |
| SampleVue | 127ms | **77ms** | **1.65x** |
| SampleRuby | 55ms | **27.9ms** | **1.97x** |

**~/lab example tool (1332 files):**

| Metric | Baseline | After opt 2 | Speedup |
|---|---|---|---|
| Avg time | 63.4ms | **41.0ms** | **1.55x** |
| Avg ratio | 39.8% | **39.8%** | unchanged |
| .java (16.5KB) | 401ms | **188ms** | **2.1x** |
| .go (4.8KB) | 131ms | **81ms** | **1.6x** |
| .vue (5.3KB) | 149ms | **103ms** | **1.45x** |
| .rb (5.6KB) | 109ms | **93ms** | **1.17x** |

**Cumulative vs original baseline:**

| Benchmark | Original | Current | Total speedup |
|---|---|---|---|
| Encode4KB | 105ms | **23.5ms** | **4.5x** |
| ~/lab avg | 63.4ms | **41.0ms** | **1.55x** |
| ~/lab .java | 401ms | **188ms** | **2.1x** |

## Failed approaches

### sync.Pool for position slices
Pool acquire/release overhead exceeded savings for small slices. Caused
26% slowdown (63.4ms → 81.4ms). Reverted.

### Early termination at minSavings=3
Slightly hurt compression ratio (-0.3%) without enough iteration savings
to offset per-iteration cost. Reverted.
