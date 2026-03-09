# boardgame

ASCII source code compression library in Go.

Boardgame compresses text in the printable ASCII range (`0x20`–`0x7E`)
using two complementary techniques:

1. **Table substitution** — repeated sequences are stored in a
   dictionary and replaced with single-byte (or 3-byte extended)
   references. Up to 255 entries; slots can be freed and reused
   mid-stream.
2. **7-bit packing** — since ASCII bytes never use bit 7, each byte is
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

Measured across a mixed-language monorepo (~1300 text files, files <= 20KB):

```
Avg Compression Time vs Avg File Size (sorted by size)
================================================================================
.tab                 3B | ######                                   6us
.jpg                15B | #####                                    5us
.png                16B | ######                                   6us
.gitattributes      19B | ########                                 11us
.mf                 25B | ######                                   7us
.tfvars             25B | #########                                15us
.timestamp          48B | ########                                 11us
.list               62B | ###########                              28us
.dockerignore       71B | #############                            51us
.bin                99B | ##################                       224us
.pub               146B | #############                            60us
.properties        156B | ###############                          89us
.tag               177B | #############                            62us
.tfstate           182B | ################                         142us
.gitignore         182B | ####################                     419us
.editorconfig      216B | ########################                 1.6ms
.expr              216B | ################                         146us
.backend           229B | ###################                      288us
.conf              229B | ####################                     405us
.example           276B | ####################                     482us
.service           334B | ###################                      372us
.cfg               378B | ######################                   742us
.pro               384B | ########################                 1.7ms
.j2                417B | #######################                  1.0ms
.json              424B | #######################                  1.3ms
.toml              560B | #########################                1.9ms
.tf                610B | #######################                  1.2ms
.mod               634B | #########################                1.8ms
.yml               680B | #########################                2.1ms
.kts               817B | ###########################              3.5ms
.code-workspace     926B | #####################                    669us
.yaml              933B | #########################                2.1ms
.xml               938B | ##########################               2.9ms
.svg              1.0KB | ###########################              4.1ms
.ts               1.1KB | ###############################          10.6ms
.d                1.2KB | #########################                1.8ms
.hcl              1.3KB | ##############################           8.3ms
.rego             1.3KB | ###########################              3.5ms
.html             1.4KB | ##############################           9.3ms
.sh               1.9KB | ################################         18.0ms
.rs               2.0KB | #############################            6.4ms
.js               2.1KB | ################################         15.7ms
.mjs              2.8KB | ##################################       26.2ms
.md               3.3KB | ##################################       32.5ms
.kt               3.5KB | #################################        18.8ms
.lock             3.8KB | ###############################          13.1ms
.bat              4.1KB | ###################################      38.4ms
.css              4.1KB | ####################################     48.2ms
.sum              4.1KB | ##################################       32.4ms
.txt              4.8KB | ###################################      39.3ms
.go               4.8KB | ####################################     58.3ms
.vue              5.3KB | #####################################    73.4ms
.rb               5.6KB | ####################################     53.9ms
.java            16.5KB | ######################################## 150.2ms

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
.expr              216B | ###############                          38.0%
.backend           229B | ##########                               27.1%
.conf              229B | ##########                               27.0%
.example           276B | #######                                  19.2%
.service           334B | ########                                 21.7%
.cfg               378B | #########                                24.3%
.pro               384B | ############                             32.3%
.j2                417B | ################                         40.2%
.json              424B | ############                             30.9%
.toml              560B | #############                            33.7%
.tf                610B | ###############                          38.7%
.mod               634B | ############                             30.1%
.yml               680B | ##############                           35.7%
.kts               817B | ################                         40.2%
.code-workspace     926B | #########################                64.3%
.yaml              933B | ######################                   57.1%
.xml               938B | #################                        43.5%
.svg              1.0KB | #############                            33.8%
.ts               1.1KB | ###############                          37.6%
.d                1.2KB | #############################            74.1%
.hcl              1.3KB | #########                                23.4%
.rego             1.3KB | #######################                  58.1%
.html             1.4KB | ##############                           36.6%
.sh               1.9KB | #############                            33.4%
.rs               2.0KB | #################                        42.8%
.js               2.1KB | #################                        44.9%
.mjs              2.8KB | ##################                       46.3%
.md               3.3KB | #############                            33.4%
.kt               3.5KB | ######################                   56.3%
.lock             3.8KB | ########################                 61.2%
.bat              4.1KB | #################                        44.4%
.css              4.1KB | ###################                      47.6%
.sum              4.1KB | ##############                           36.3%
.txt              4.8KB | ##########                               26.7%
.go               4.8KB | ###################                      48.3%
.vue              5.3KB | ################                         41.8%
.rb               5.6KB | #####################                    53.8%
.java            16.5KB | ##############################           75.1%
```

Overall: **39.8% average compression** across 1332 files. Compression time
scales superlinearly with file size; ratio improves with size as larger
files contain more repeated patterns.

### Comparison with standard compressors

Measured on the same file set (1333 files, max 20KB per file):

| Algorithm | Avg Ratio | Avg Time | Throughput |
|-----------|----------|----------|------------|
| brotli | 53.5% | 1.7ms | 1.6 MB/s |
| zstd | 43.2% | 7.2ms | 386 KB/s |
| **boardgame** | **39.9%** | **27.7ms** | **101 KB/s** |
| gzip | 39.8% | 684us | 4.0 MB/s |
| snappy | 37.6% | 40us | 68.3 MB/s |
| lz4 | 24.8% | 1.8ms | 1.5 MB/s |

Boardgame matches gzip's ratio overall but is ~40x slower due to the
suffix-array candidate search. Its advantage is on **small files** (< 500B)
where standard compressors' fixed headers hurt: boardgame compresses
`.gitignore` (182B) at 22% vs gzip's -12%, `.json` (425B) at 31% vs gzip's
25%, and `.yml` (680B) at 36% vs gzip's -2%. The 7-bit packing layer
provides a guaranteed ~12.5% saving that other compressors cannot match on
sub-kilobyte ASCII inputs.

On larger files (> 2KB), standard LZ77-based compressors outperform
boardgame's dictionary substitution: gzip reaches 65% on `.go` files vs
boardgame's 48%.

Full comparison data: [example/compare/README.md](example/compare/README.md).

## Testing

```
go test ./...                          # unit tests
go test -bench=. -benchmem             # benchmarks
go test -fuzz=FuzzRoundTrip -fuzztime=30s  # fuzz: encode/decode round-trip
go test -fuzz=FuzzDecode -fuzztime=30s     # fuzz: decode crash resistance
```
