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

Measured across a mixed-language monorepo (~1100 text files, files <= 20KB):

```
Avg Compression Time vs Avg File Size (sorted by size)
================================================================================
.tab                 3B | ####                                     4us
.jpg                15B | ####                                     5us
.png                16B | ####                                     4us
.gitattributes      19B | ######                                   8us
.mf                 25B | #########                                25us
.tfvars             25B | ######                                   8us
.timestamp          48B | ########                                 16us
.list               62B | #################                        326us
.dockerignore       71B | #############                            68us
.pub               146B | ############                             50us
.properties        156B | #################                        251us
.tag               177B | ##############                           100us
.tfstate           182B | ###############                          144us
.gitignore         182B | ###################                      487us
.conf              213B | ####################                     878us
.expr              216B | ###################                      476us
.editorconfig      216B | ###################                      607us
.backend           229B | ####################                     787us
.example           276B | ####################                     681us
.service           334B | ####################                     829us
.cfg               378B | ######################                   1.3ms
.pro               384B | #######################                  2.0ms
.j2                417B | #######################                  2.1ms
.json              423B | ########################                 2.5ms
.toml              560B | ########################                 3.1ms
.tf                610B | ########################                 2.7ms
.mod               634B | #########################                3.5ms
.yml               680B | ##########################               4.8ms
.sh                763B | ##########################               4.8ms
.kts               817B | ###########################              6.8ms
.html              881B | ############################             8.8ms
.xml               918B | ###########################              8.0ms
.code-workspace     926B | ####################                     671us
.yaml              928B | ##########################               5.2ms
.svg              1.0KB | ############################             9.0ms
.ts               1.1KB | ##############################           20.5ms
.d                1.2KB | ############################             10.2ms
.hcl              1.3KB | #############################            15.4ms
.rego             1.3KB | ############################             9.2ms
.js               1.8KB | ###############################          25.0ms
.rs               2.5KB | ##############################           20.2ms
.mjs              2.8KB | #################################        50.7ms
.md               3.1KB | ##################################       62.1ms
.kt               3.5KB | #################################        47.2ms
.lock             3.8KB | ###############################          30.4ms
.bat              4.1KB | ###################################      103.0ms
.css              4.1KB | ###################################      102.1ms
.sum              4.1KB | ##################################       67.6ms
.go               4.5KB | ####################################     118.8ms
.txt              4.8KB | ##################################       64.1ms
.vue              4.9KB | ####################################     133.9ms
.rb               5.6KB | ####################################     116.2ms
.java            16.5KB | ######################################## 414.6ms

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
.pub               146B | #####                                    13.0%
.properties        156B | ##########                               26.0%
.tag               177B | #######                                  19.8%
.tfstate           182B | ##########                               25.8%
.gitignore         182B | ########                                 21.7%
.conf              213B | #########                                24.6%
.expr              216B | ###############                          38.0%
.editorconfig      216B | ########                                 21.8%
.backend           229B | ##########                               27.1%
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
.sh                763B | ##############                           35.9%
.kts               817B | ################                         40.3%
.html              881B | ###########                              29.2%
.xml               918B | #################                        43.0%
.code-workspace     926B | #########################                64.3%
.yaml              928B | ######################                   57.1%
.svg              1.0KB | #############                            33.8%
.ts               1.1KB | ##############                           37.3%
.d                1.2KB | #############################            74.1%
.hcl              1.3KB | #########                                23.4%
.rego             1.3KB | #######################                  58.1%
.js               1.8KB | #################                        44.4%
.rs               2.5KB | ####################                     51.3%
.mjs              2.8KB | ##################                       46.3%
.md               3.1KB | #############                            34.7%
.kt               3.5KB | ######################                   56.3%
.lock             3.8KB | ########################                 61.2%
.bat              4.1KB | #################                        44.4%
.css              4.1KB | ###################                      47.6%
.sum              4.1KB | ##############                           36.3%
.go               4.5KB | ###################                      48.8%
.txt              4.8KB | ##########                               26.7%
.vue              4.9KB | ################                         40.8%
.rb               5.6KB | #####################                    53.7%
.java            16.5KB | ##############################           75.0%
```

Overall: **40.0% average compression** across 1133 files. Compression time
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
