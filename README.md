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
go get boardgame
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
| Encode 100B | 100 B | 12.5 MB/s | 22 |
| Encode 1KB | 1 KB | 0.18 MB/s | 46K |
| Encode 4KB | 4 KB | 0.04 MB/s | 553K |
| Encode 16KB | 16 KB | 0.01 MB/s | 2.6M |
| Decode 1KB | 1 KB | 199 MB/s | 13 |
| Decode 16KB | 16 KB | 868 MB/s | 22 |
| Pack7 4KB | 4 KB | 39 MB/s | 14 |
| Unpack7 4KB | 4 KB | 60 MB/s | 12 |

Decoding is ~5000x faster than encoding. The encoder is CPU-bound in the
suffix-array-based candidate search; the decoder and 7-bit packing layers
are fast.

### Compression ratio by file type

Measured across a mixed-language monorepo (~1300 text files, files <= 20KB):

```
Avg Compression Time vs Avg File Size (sorted by size)
================================================================================
.tab                 3B | ###                                      3us
.jpg                15B | ######                                   8us
.png                16B | ####                                     5us
.gitattributes      19B | #######                                  13us
.mf                 25B | #######                                  12us
.tfvars             25B | ########                                 17us
.timestamp          48B | ########                                 17us
.list               62B | ##########                               34us
.dockerignore       71B | ###############                          159us
.bin                99B | ############                             58us
.pub               146B | #############                            74us
.properties        156B | #################                        294us
.tag               177B | ##############                           97us
.tfstate           182B | ###############                          167us
.gitignore         182B | ##################                       467us
.editorconfig      216B | ###################                      591us
.expr              216B | #################                        255us
.backend           229B | ########################                 2.5ms
.conf              229B | ###################                      623us
.example           276B | #####################                    1.0ms
.service           334B | ####################                     838us
.cfg               378B | #######################                  2.0ms
.pro               384B | #########################                3.7ms
.j2                417B | #######################                  1.9ms
.json              423B | ########################                 2.4ms
.toml              560B | #########################                3.4ms
.tf                610B | #########################                3.4ms
.mod               634B | #########################                3.7ms
.yml               680B | ##########################               5.0ms
.kts               817B | ##########################               5.9ms
.code-workspace     926B | ######################                   1.5ms
.yaml              928B | ##########################               5.0ms
.xml               938B | ###########################              7.4ms
.svg              1.0KB | ############################             8.8ms
.ts               1.1KB | ##############################           22.4ms
.d                1.2KB | ############################             10.1ms
.hcl              1.3KB | #############################            15.2ms
.rego             1.3KB | ###########################              7.3ms
.html             1.4KB | ##############################           17.4ms
.sh               1.9KB | ################################         32.4ms
.js               2.1KB | ################################         32.4ms
.rs               2.5KB | ###############################          25.5ms
.mjs              2.8KB | ##################################       63.0ms
.md               3.3KB | ##################################       60.5ms
.kt               3.5KB | ################################         42.0ms
.txt              3.8KB | #################################        48.3ms
.lock             3.8KB | ###############################          28.0ms
.bat              4.1KB | ###################################      91.6ms
.css              4.1KB | ###################################      104.6ms
.sum              4.1KB | ##################################       68.1ms
.go               4.8KB | ####################################     133.9ms
.vue              5.3KB | ####################################     151.9ms
.rb               5.6KB | ###################################      99.0ms
.java            16.5KB | ######################################## 418.1ms

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
.service           334B | ########                                 21.8%
.cfg               378B | #########                                24.3%
.pro               384B | ############                             32.3%
.j2                417B | ################                         40.2%
.json              423B | ############                             30.8%
.toml              560B | #############                            33.2%
.tf                610B | ###############                          38.9%
.mod               634B | ############                             30.0%
.yml               680B | ##############                           35.7%
.kts               817B | ################                         40.3%
.code-workspace     926B | #########################                64.3%
.yaml              928B | ######################                   57.1%
.xml               938B | #################                        43.5%
.svg              1.0KB | #############                            33.8%
.ts               1.1KB | ###############                          37.6%
.d                1.2KB | #############################            74.1%
.hcl              1.3KB | #########                                23.4%
.rego             1.3KB | #######################                  58.1%
.html             1.4KB | ##############                           36.5%
.sh               1.9KB | #############                            33.4%
.js               2.1KB | #################                        45.0%
.rs               2.5KB | ####################                     51.3%
.mjs              2.8KB | ##################                       46.3%
.md               3.3KB | #############                            33.3%
.kt               3.5KB | ######################                   56.3%
.txt              3.8KB | ##########                               26.7%
.lock             3.8KB | ########################                 61.2%
.bat              4.1KB | #################                        44.4%
.css              4.1KB | ###################                      47.6%
.sum              4.1KB | ##############                           36.3%
.go               4.8KB | ###################                      48.4%
.vue              5.3KB | ################                         41.7%
.rb               5.6KB | #####################                    53.7%
.java            16.5KB | ##############################           75.0%
```

Overall: **39.8% average compression** across 1325 files. Compression time
scales superlinearly with file size; ratio improves with size as larger
files contain more repeated patterns.

### Future optimization

The encoder rebuilds the suffix array from scratch on each greedy
iteration (up to 255 rounds). The `buildSA` function currently uses
`sort.Slice`, which allocates comparison closures and is O(m log m) per
call. This accounts for most of the 553K allocations at 4KB. Possible
improvements:

- **Reuse the suffix array** across iterations instead of rebuilding,
  updating it incrementally as substitutions replace substrings with
  reference bytes.
- **Use SA-IS or DC3/skew** algorithms for O(n) suffix array construction
  instead of comparison-based sorting.
- **Pool allocations** for the SA, LCP, and position slices to reduce GC
  pressure.

## Testing

```
go test ./...                          # unit tests
go test -bench=. -benchmem             # benchmarks
go test -fuzz=FuzzRoundTrip -fuzztime=30s  # fuzz: encode/decode round-trip
go test -fuzz=FuzzDecode -fuzztime=30s     # fuzz: decode crash resistance
```
