# Compression algorithm comparison

## Setup

Boardgame vs five standard compressors on the same file set (~/lab
monorepo, Intel i7-8700 @ 3.20GHz). Two runs: default 20KB cap and
unlimited.

Algorithms: gzip (stdlib DEFLATE), snappy (speed-first LZ77), zstd
(modern balanced), lz4 (extreme speed), brotli (text-optimized high
ratio).

## Results: 20KB cap (1333 files, 3.6MB)

| Algorithm | Avg Ratio | Avg Time | Throughput |
|-----------|----------|----------|------------|
| brotli | 53.5% | 1.7ms | 1.6 MB/s |
| zstd | 43.2% | 7.2ms | 386 KB/s |
| boardgame | 39.9% | 27.7ms | 101 KB/s |
| gzip | 39.8% | 684us | 4.0 MB/s |
| snappy | 37.6% | 40us | 68.3 MB/s |
| lz4 | 24.8% | 1.8ms | 1.5 MB/s |

## Results: unlimited file size (1396 files, 7.5MB)

| Algorithm | Avg Ratio | Avg Time | Throughput |
|-----------|----------|----------|------------|
| brotli | 54.7% | 1.4ms | 3.8 MB/s |
| zstd | 44.8% | 4.9ms | 1.1 MB/s |
| gzip | 41.6% | 492us | 10.9 MB/s |
| boardgame | 40.8% | 115.7ms | 47 KB/s |
| snappy | 39.0% | 34us | 156.2 MB/s |
| lz4 | 26.8% | 1.2ms | 4.5 MB/s |

### Large file encode times (boardgame)

| Extension | Avg Size | Avg Time |
|-----------|---------|----------|
| .sarif | 115.7KB | 2.51s |
| .d | 28.7KB | 1.04s |
| .js | 22.0KB | 1.02s |
| .backup | 22.3KB | 245ms |
| .java | 18.2KB | 141ms |
| .json | 7.2KB | 150ms |
| .css | 8.0KB | 131ms |
| .vue | 7.1KB | 114ms |
| .go | 6.6KB | 93ms |

## Analysis

### Boardgame wins on small files (< 500B)

Standard compressors add fixed-size headers (gzip: 18B, zstd: ~13B,
lz4: 15B) that dominate on tiny inputs. Boardgame's format has no fixed
header — just inline table definitions — and the 7-bit packing layer
provides a guaranteed ~12.5% saving unconditionally.

Selected small-file ratios (20KB cap run):

| Extension | Avg Size | boardgame | gzip | zstd | lz4 |
|-----------|---------|-----------|------|------|-----|
| .gitignore | 182B | 22% | -12% | 2% | -21% |
| .json | 425B | 31% | 25% | 28% | 8% |
| .yml | 680B | 36% | -2% | 17% | -8% |
| .mod | 641B | 31% | 5% | 15% | -3% |
| .timestamp | 48B | 10% | -46% | -27% | -40% |
| .properties | 156B | 26% | -2% | 6% | -7% |

### Standard compressors win on larger files (> 2KB)

LZ77 back-references amortize header overhead and outperform dictionary
table substitution as file size grows.

Selected large-file ratios (unlimited run):

| Extension | Avg Size | boardgame | gzip | zstd | brotli |
|-----------|---------|-----------|------|------|--------|
| .go | 6.6KB | 49% | 66% | 64% | 69% |
| .java | 18.2KB | 76% | 89% | 88% | 90% |
| .js | 22.0KB | 45% | 54% | 54% | 61% |
| .vue | 7.1KB | 42% | 61% | 59% | 66% |
| .sarif | 115.7KB | 77% | 89% | 88% | 89% |

### Boardgame scales poorly with file size

The suffix-array candidate search runs once per greedy iteration (~25
iterations per file). Each rebuild is O(n log n) for SA construction
plus O(n) for the LCP scan, giving roughly O(n² log n) total encoder
cost. This is acceptable for small files but becomes a bottleneck above
~10KB:

- 1KB file: ~1ms encode
- 5KB file: ~50ms encode
- 20KB file: ~1s encode
- 100KB file: ~2.5s encode

Standard compressors use O(n) sliding-window approaches and barely
change speed as files grow. Snappy compresses a 116KB file in
microseconds; boardgame takes 2.5 seconds.

### Removing the file size cap

With the 20KB cap removed:
- 64 additional files are included (1333 → 1396)
- Total data grows from 3.6MB to 7.5MB
- Boardgame avg time jumps from 28ms to 116ms (4.1x slower)
- Gzip avg time drops slightly (684us → 492us) due to better
  amortization of per-file overhead
- Gzip overtakes boardgame on overall ratio (41.6% vs 40.8%)
- No crashes or errors — boardgame handles large files correctly

## Conclusion

Boardgame works on files of any size but is designed for small ASCII
text files (< 5KB). In this range it matches or beats gzip/zstd on
compression ratio while requiring no format header. Above ~10KB,
standard compressors are both faster and achieve better ratios.

The recommended `-max-size 20480` default in the example tools reflects
this: it excludes files where boardgame is not competitive.
