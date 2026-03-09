# boardgame comparison tool

Compresses every text source file in a directory tree with boardgame and
five standard compression algorithms, reporting side-by-side ratios,
timing, and throughput.

## Algorithms

| Algorithm | Package | Type |
|-----------|---------|------|
| boardgame | `git.risottobias.org/claude/boardgame` | this library |
| gzip | `compress/gzip` (stdlib) | DEFLATE, industry standard |
| snappy | `github.com/golang/snappy` | speed-optimized LZ77 |
| zstd | `github.com/klauspost/compress/zstd` | modern balanced compressor |
| lz4 | `github.com/pierrec/lz4/v4` | extreme-speed LZ77 |
| brotli | `github.com/andybalholm/brotli` | text-optimized, high ratio |

## Usage

```
go run ./example/compare /path/to/project
go run ./example/compare -exclude vendor /path/to/project
go run ./example/compare -include-vendored -max-size 0 -workers 8 /path/to/project
```

Flags:
- `-exclude` — additional directory name to skip
- `-include-vendored` — include `node_modules` and `vendor` (excluded by default)
- `-max-size` — maximum file size in bytes (default 20KB, 0 = unlimited)
- `-workers` — parallel compression workers (default: number of CPUs)

## Results

Ran against a mixed-language monorepo (1333 text files, max 20KB per file,
Intel i7-8700 @ 3.20GHz):

### Overall summary

| Algorithm | Avg Ratio | Total Out | Avg Time | Throughput |
|-----------|----------|-----------|----------|------------|
| boardgame | 39.9% | 1.9MB | 27.7ms | 101 KB/s |
| gzip | 39.8% | 1.2MB | 684us | 4.0 MB/s |
| snappy | 37.6% | 1.6MB | 40us | 68.3 MB/s |
| zstd | 43.2% | 1.2MB | 7.2ms | 386 KB/s |
| lz4 | 24.8% | 1.8MB | 1.8ms | 1.5 MB/s |
| brotli | 53.5% | 1.1MB | 1.7ms | 1.6 MB/s |

### Avg compression ratio by extension (%)

```
Extension    Files  AvgSize  boardgame       gzip     snappy       zstd        lz4     brotli
---------------------------------------------------------------------------------------------
.bat             1    4.1KB      44.4%      64.6%      50.7%      62.2%      46.1%      67.5%
.cfg             3     378B      24.3%      26.5%       7.9%      28.0%      -0.3%      36.1%
.code-workspace  1     926B      64.3%      67.7%      55.1%      66.7%      52.5%      68.9%
.css            13    4.1KB      47.6%      61.5%      46.2%      59.1%      41.0%      66.5%
.d              18    1.2KB      74.1%      76.0%      71.0%      76.5%      69.6%      80.1%
.go            368    4.8KB      48.3%      65.1%      51.9%      63.0%      47.3%      67.6%
.html           55    1.4KB      36.6%      46.7%      31.4%      45.5%      25.2%      60.5%
.java            3   16.5KB      75.1%      88.2%      81.4%      86.9%      81.0%      89.0%
.js             46    2.1KB      44.9%      51.3%      42.8%      51.8%      34.9%      59.2%
.json           98     425B      30.8%      24.8%      20.5%      27.9%       7.8%      38.5%
.kt             58    3.5KB      56.3%      69.3%      58.4%      67.8%      54.1%      71.8%
.md            160    3.3KB      33.5%      50.4%      34.7%      48.9%      26.8%      56.0%
.mod            33     641B      30.6%       4.5%      18.5%      15.2%      -3.4%      30.4%
.vue            96    5.3KB      41.8%      59.9%      46.1%      57.9%      40.5%      65.1%
.xml            28     938B      43.5%      45.9%      36.3%      46.5%      30.6%      59.4%
.yaml           44     933B      57.1%      66.6%      56.4%      65.8%      50.8%      71.0%
.yml            73     680B      35.7%      -2.0%      28.3%      16.6%      -8.2%      39.1%
```

(Selected extensions shown; full output includes all 55 extensions.)

### Avg compression time by extension

```
Extension    Files  AvgSize  boardgame       gzip     snappy       zstd        lz4     brotli
---------------------------------------------------------------------------------------------
.go            368    4.8KB     54.9ms      832us       49us      6.3ms      1.5ms      1.8ms
.java            3   16.5KB    148.4ms      480us       78us      7.8ms      1.4ms      1.5ms
.json           98     425B      1.0ms      535us       17us      7.6ms      2.2ms      1.5ms
.kt             58    3.5KB     19.1ms      1.1ms       44us      6.2ms      1.9ms      1.5ms
.md            160    3.3KB     29.6ms      707us       31us      7.6ms      1.9ms      2.2ms
.vue            96    5.3KB     70.2ms      739us       73us      7.6ms      1.7ms      1.9ms
.yml            73     680B      1.8ms      749us       20us      8.9ms      2.1ms      1.6ms
```

(Selected extensions shown.)

### Observations

- **Boardgame matches gzip's overall ratio** (39.9% vs 39.8%) but is ~40x
  slower. The encoder's suffix-array candidate search is CPU-bound;
  standard compressors use faster LZ77 sliding windows.

- **Boardgame wins on small files** (< 500B). Standard compressors add
  fixed-size headers (gzip: 18B, zstd: ~13B, lz4: 15B) that dominate on
  tiny inputs. Boardgame's format has no fixed header, just inline table
  definitions. Examples:
  - `.gitignore` (182B avg): boardgame 22% vs gzip -12%, zstd 2%, lz4 -21%
  - `.json` (425B avg): boardgame 31% vs gzip 25%, lz4 8%
  - `.yml` (680B avg): boardgame 36% vs gzip -2%, lz4 -8%
  - `.mod` (641B avg): boardgame 31% vs gzip 5%, zstd 15%

- **Standard compressors win on larger files** (> 2KB). With more data,
  LZ77 back-references amortize their overhead and outperform
  dictionary-based table substitution:
  - `.go` (4.8KB avg): gzip 65% vs boardgame 48%
  - `.java` (16.5KB avg): gzip 88% vs boardgame 75%
  - `.vue` (5.3KB avg): gzip 60% vs boardgame 42%

- **Brotli achieves the best ratio** (53.5%) across the board. It was
  designed for text compression and uses a pre-built static dictionary
  of common web/text patterns.

- **Snappy is the speed champion** — 68 MB/s throughput, ~700x faster than
  boardgame — but compresses the least (37.6%).

- **Zstd is the best all-rounder** among standard compressors: better ratio
  than gzip (43.2% vs 39.8%) at reasonable speed.

- **Boardgame's niche** is small ASCII text files where format overhead
  matters and the 7-bit packing layer provides a guaranteed ~12.5% saving
  that other compressors cannot match on sub-kilobyte inputs.
