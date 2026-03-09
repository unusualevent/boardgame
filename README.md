# boardgame

ASCII source code compression library in Go.

Boardgame compresses text in the printable ASCII range (`0x20`–`0x7E`)
using three complementary techniques:

1. **RLE** — runs of 4–15 spaces or tabs are collapsed to 2 bytes,
   shrinking whitespace-heavy input before the suffix array runs.
2. **Table substitution** — repeated sequences are stored in a
   dictionary and replaced with single-byte (or 3-byte extended)
   references. Up to 251 entries (4 byte values are reserved); slots
   can be freed and reused mid-stream.
3. **7-bit packing** — since ASCII bytes never use bit 7, each byte is
   stored as 7 bits, saving ~12.5% unconditionally.

## Install

```
go get git.risottobias.org/claude/boardgame
```

## Quick start

```go
compressed, err := boardgame.Encode([]byte("hello hello hello"))
restored, err := boardgame.Decode(compressed)
```

## API

| Function | Description |
|----------|-------------|
| `Encode(src []byte) ([]byte, error)` | Compress source bytes |
| `Decode(src []byte) ([]byte, error)` | Decompress encoded bytes |
| `Stats(orig, comp []byte) (int, int, float64)` | Original size, compressed size, ratio |

Input can contain any bytes. Printable ASCII (`0x20`–`0x7E`), tab, and
newline participate in compression; non-ASCII bytes (UTF-8, etc.) are
DEL-escaped and pass through transparently. See [docs/api.md](docs/api.md)
for full details.

## Example

The `example/` directory contains a CLI tool that walks a directory tree,
compresses each text source file, and reports per-extension compression
ratios, timing, and ASCII histograms. See [example/README.md](example/README.md).

```
go run ./example /path/to/project
go run ./example -exclude testdata -max-size 0 /path/to/project
```

A comparison tool benchmarks boardgame against gzip, snappy, zstd, lz4,
and brotli on the same file set. See
[example/compare/README.md](example/compare/README.md).

```
go run ./example/compare /path/to/project
```

## Documentation

- [docs/format.md](docs/format.md) — wire format specification
- [docs/api.md](docs/api.md) — API reference and usage examples
- [docs/internals.md](docs/internals.md) — compression pipeline and
  algorithm details

## Performance

### Benchmarks

```
go test -bench=. -benchmem
```

Encoding throughput on an Intel i7-8700 @ 3.20GHz:

| Benchmark | Size | Throughput | Allocs/op |
|---|---|---|---|
| Encode 100B | 100 B | 12.5 MB/s | 64 |
| Encode 1KB | 1 KB | 1.14 MB/s | 126 |
| Encode 4KB | 4 KB | 0.15 MB/s | 349 |
| Encode 16KB | 16 KB | 0.02 MB/s | 147 |
| Decode 1KB | 1 KB | 236 MB/s | 15 |
| Decode 16KB | 16 KB | 938 MB/s | 24 |
| Pack7 4KB | 4 KB | 39 MB/s | 14 |
| Unpack7 4KB | 4 KB | 63 MB/s | 12 |

Decoding is ~4000x faster than encoding. The encoder is CPU-bound in the
suffix-array-based candidate search; the decoder and 7-bit packing layers
are fast.

### Compression ratio by file type

Measured across a mixed-language monorepo (~1335 text files, files <= 20KB):

```
Avg Compression Time vs Avg File Size (sorted by size)
================================================================================
.tab                 3B | #########                                20us
.jpg                15B | ######                                   7us
.png                16B | ####                                     4us
.gitattributes      19B | ######                                   7us
.mf                 25B | #########                                17us
.tfvars             25B | ######                                   8us
.timestamp          48B | ##################                       332us
.list               62B | #########                                20us
.dockerignore       71B | ############                             43us
.bin                99B | #########                                20us
.pub               146B | ##############################           12.4ms
.properties        156B | ######################                   938us
.tag               177B | ##############                           79us
.tfstate           182B | ##############                           80us
.gitignore         182B | #####################                    837us
.editorconfig      216B | ################                         184us
.expr              216B | ##############                           100us
.backend           229B | #################                        232us
.conf              229B | ################                         189us
.example           276B | #####################                    831us
.service           334B | #####################                    787us
.cfg               378B | ####################                     599us
.pro               384B | #########################                2.4ms
.j2                417B | ############################             7.4ms
.json              425B | ########################                 2.2ms
.toml              560B | ############################             6.8ms
.tf                610B | ###########################              4.5ms
.mod               641B | ##########################               4.0ms
.yml               680B | ###########################              4.9ms
.kts               817B | #########################                2.7ms
.code-workspace     926B | ######################                   929us
.yaml              933B | ##########################               3.2ms
.xml               938B | ###########################              4.6ms
.svg              1.0KB | ##############################           14.4ms
.ts               1.1KB | ###############################          19.1ms
.d                1.2KB | #######################                  1.6ms
.hcl              1.3KB | ################################         26.9ms
.rego             1.3KB | ##########################               4.0ms
.html             1.4KB | ##############################           11.7ms
.sh               1.9KB | #################################        27.9ms
.rs               2.0KB | ##############################           14.4ms
.js               2.1KB | ################################         23.8ms
.mjs              2.8KB | ###################################      54.6ms
.md               3.4KB | ###################################      55.6ms
.kt               3.5KB | #################################        30.9ms
.lock             3.8KB | #############################            8.8ms
.sum              4.0KB | ##################################       50.6ms
.bat              4.1KB | ######################################   138.5ms
.css              4.1KB | ###################################      62.0ms
.txt              4.8KB | ####################################     88.9ms
.go               4.8KB | #####################################    97.5ms
.vue              5.3KB | #####################################    123.7ms
.rb               5.6KB | ##################################       47.6ms
.java            16.5KB | ######################################## 240.2ms

Avg Compression Ratio vs Avg File Size (sorted by size)
================================================================================
.tab                 3B |                                          -33.3%
.jpg                15B | #                                        4.5%
.png                16B | ##                                       6.2%
.gitattributes      19B | ##                                       5.3%
.mf                 25B |                                          0.0%
.tfvars             25B | ###                                      8.0%
.timestamp          48B | ####                                     10.4%
.list               62B | ####                                     11.6%
.dockerignore       71B | #######                                  19.7%
.bin                99B |                                          -31.3%
.pub               146B | #####                                    13.0%
.properties        156B | ##########                               26.0%
.tag               177B | #######                                  19.8%
.tfstate           182B | ##########                               25.8%
.gitignore         182B | ########                                 21.7%
.editorconfig      216B | ########                                 21.8%
.expr              216B | ################                         40.3%
.backend           229B | ##########                               27.1%
.conf              229B | ##########                               27.4%
.example           276B | #######                                  19.7%
.service           334B | ########                                 21.4%
.cfg               378B | #########                                23.8%
.pro               384B | ############                             32.3%
.j2                417B | ################                         40.3%
.json              425B | ###########                              29.6%
.toml              560B | #############                            33.9%
.tf                610B | ###############                          39.1%
.mod               641B | ############                             30.7%
.yml               680B | #############                            34.4%
.kts               817B | ################                         40.3%
.code-workspace     926B | #########################                64.3%
.yaml              933B | ######################                   56.0%
.xml               938B | #################                        43.9%
.svg              1.0KB | #############                            34.0%
.ts               1.1KB | ###############                          37.8%
.d                1.2KB | #############################            74.1%
.hcl              1.3KB | #########                                23.8%
.rego             1.3KB | #######################                  58.1%
.html             1.4KB | ##############                           36.0%
.sh               1.9KB | #############                            33.4%
.rs               2.0KB | ################                         41.5%
.js               2.1KB | #################                        44.9%
.mjs              2.8KB | ##################                       46.0%
.md               3.4KB | #############                            34.7%
.kt               3.5KB | ######################                   56.8%
.lock             3.8KB | ########################                 61.6%
.sum              4.0KB | ##############                           36.4%
.bat              4.1KB | ##################                       45.2%
.css              4.1KB | ###################                      47.8%
.txt              4.8KB | ##########                               26.7%
.go               4.8KB | ###################                      49.2%
.vue              5.3KB | #################                        43.0%
.rb               5.6KB | ####################                     50.3%
.java            16.5KB | #############################            72.6%
```

Overall: **40.1% average compression** across 1335 files. Compression time
scales superlinearly with file size; ratio improves with size as larger
files contain more repeated patterns.

### Comparison with standard compressors

Measured on the same file set (1335 files, max 20KB per file):

| Algorithm | Avg Ratio | Avg Time | Throughput |
|-----------|----------|----------|------------|
| brotli | 53.5% | 2.3ms | 1.2 MB/s |
| zstd | 43.2% | 10.0ms | 281 KB/s |
| **boardgame** | **40.1%** | **37.4ms** | **75 KB/s** |
| gzip | 39.8% | 1.1ms | 2.6 MB/s |
| snappy | 37.6% | 42us | 63.9 MB/s |
| lz4 | 24.8% | 1.9ms | 1.5 MB/s |

Boardgame beats gzip's ratio (40.1% vs 39.8%) thanks to RLE preprocessing
that frees table slots for higher-value patterns, but is ~34x slower due to
the suffix-array candidate search. Its advantage is on **small files** (< 500B)
where standard compressors' fixed headers hurt: boardgame compresses
`.gitignore` (182B) at 22% vs gzip's -12%, `.json` (425B) at 30% vs gzip's
25%, and `.yml` (680B) at 34% vs gzip's -2%. The 7-bit packing layer
provides a guaranteed ~12.5% saving that other compressors cannot match on
sub-kilobyte ASCII inputs.

On larger files (> 2KB), standard LZ77-based compressors outperform
boardgame's dictionary substitution: gzip reaches 65% on `.go` files vs
boardgame's 49%.

Full comparison data: [example/compare/README.md](example/compare/README.md).

## Testing

```
go test ./...                          # unit tests
go test -bench=. -benchmem             # benchmarks
go test -fuzz=FuzzRoundTrip -fuzztime=30s  # fuzz: encode/decode round-trip
go test -fuzz=FuzzDecode -fuzztime=30s     # fuzz: decode crash resistance
```
