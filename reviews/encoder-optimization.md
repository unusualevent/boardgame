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

### 4. Radix sort for SA construction
Replace full comparison-based `sort.Slice` with a single-byte radix
partition (counting sort on first byte) followed by comparison sort
within each bucket. With ~97 distinct literal byte values, buckets
average n/97 elements, making per-bucket sorts very fast.

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

### After optimization 2: LCP interval scan (commit 5ec07a8)

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

### After optimization 4: radix sort SA construction

Single-byte radix partition (counting sort on first byte) reduces
comparison sort to within-bucket only. Each bucket averages n/97
elements, drastically reducing sort work.

**go test -bench (count=3 medians):**

| Benchmark | Before (LCP scan) | After (radix SA) | Speedup |
|---|---|---|---|
| Encode4KB | 23.5ms | **28ms** | 0.84x (slight regression) |
| RealLibrary | 16.4ms | **12.3ms** | **1.33x** |
| RealTests | 107ms | **82ms** | **1.30x** |
| SampleGo | 29ms | **21.7ms** | **1.34x** |
| SampleHTML | 14.3ms | **11.1ms** | **1.29x** |
| SampleJS | 36.5ms | **26.7ms** | **1.37x** |
| SampleJSON | 5.9ms | **4.3ms** | **1.37x** |
| SampleVue | 77ms | **58ms** | **1.33x** |
| SampleRuby | 27.9ms | **20.7ms** | **1.35x** |

**~/lab example tool (1332 files):**

| Metric | After opt 2 | After opt 4 | Speedup |
|---|---|---|---|
| Avg time | 41.0ms | **29.1ms** | **1.41x** |
| Avg ratio | 39.8% | **39.8%** | unchanged |
| .java (16.5KB) | 188ms | **147ms** | **1.28x** |
| .go (4.8KB) | 81ms | **58ms** | **1.40x** |
| .vue (5.3KB) | 103ms | **72ms** | **1.43x** |
| .rb (5.6KB) | 93ms | **57ms** | **1.63x** |

### Cumulative: original baseline → final

| Benchmark | Original | Final | Total speedup |
|---|---|---|---|
| Encode4KB | 105ms | **28ms** | **3.75x** |
| RealLibrary | 22.2ms | **12.3ms** | **1.80x** |
| RealTests | 189ms | **82ms** | **2.30x** |
| SampleGo | 51ms | **21.7ms** | **2.35x** |
| SampleHTML | 26ms | **11.1ms** | **2.34x** |
| SampleJS | 47ms | **26.7ms** | **1.76x** |
| SampleJSON | 9.6ms | **4.3ms** | **2.23x** |
| SampleVue | 112ms | **58ms** | **1.93x** |
| SampleRuby | 36ms | **20.7ms** | **1.74x** |
| ~/lab avg | 63.4ms | **29.1ms** | **2.18x** |
| ~/lab .java | 401ms | **147ms** | **2.73x** |
| ~/lab .go | 131ms | **58ms** | **2.26x** |
| ~/lab .vue | 149ms | **72ms** | **2.07x** |

## Failed approaches

### sync.Pool for position slices
Pool acquire/release overhead exceeded savings for small slices. Caused
26% slowdown (63.4ms → 81.4ms). Reverted.

### Early termination at minSavings=3
Slightly hurt compression ratio (-0.3%) without enough iteration savings
to offset per-iteration cost. Reverted.

### Batch SA build (apply multiple candidates per SA construction)
Extracted all candidates from one SA build, then applied them one at a
time with O(n) re-scoring via `countNonOverlap`. Failed for two reasons:
1. Unlimited batching corrupted the encoding (reference to undefined
   table entry errors on real files)
2. Even with safe batch limits (1-16), the O(k² * n) re-scoring cost
   exceeded the SA rebuild cost it was trying to avoid
Reverted entirely.

### sort.Interface concrete type for SA construction
Replacing `sort.Slice` closure with a concrete `sort.Interface` type
was ~20% slower. Go's pdqsort closure implementation is well-optimized;
the virtual dispatch overhead of `sort.Interface` exceeded any savings.

### Two-byte radix sort (256² = 65K buckets)
Allocating and iterating 65K-element count/offset arrays was too
expensive for typical input sizes (n < 20K). Twice as slow as the
single-byte radix approach.
