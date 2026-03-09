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
| Encode 100B | 100 B | 12.2 MB/s | 22 |
| Encode 1KB | 1 KB | 0.18 MB/s | 46K |
| Encode 4KB | 4 KB | 0.04 MB/s | 553K |
| Encode 16KB | 16 KB | 0.01 MB/s | 2.6M |
| Decode 1KB | 1 KB | 235 MB/s | 13 |
| Decode 16KB | 16 KB | 890 MB/s | 22 |
| Pack7 4KB | 4 KB | 36 MB/s | 14 |
| Unpack7 4KB | 4 KB | 59 MB/s | 12 |

Decoding is ~5000x faster than encoding. The encoder is CPU-bound in the
suffix-array-based candidate search; the decoder and 7-bit packing layers
are fast.

### Compression ratio by file type

Measured across a mixed-language monorepo (~1100 text files, files <= 20KB):

```
Avg Compression Time vs Avg File Size (sorted by size)
================================================================================
.timestamp          48B | #############                            86us
.list               62B | #########                                22us
.dockerignore       71B | ############                             49us
.pub               146B | ############                             62us
.properties        156B | ###################                      525us
.tag               177B | ##############                           88us
.gitignore         182B | ###################                      475us
.conf              213B | #################                        279us
.editorconfig      216B | ##################                       380us
.backend           229B | ######################                   1.4ms
.example           276B | #######################                  1.7ms
.service           334B | #####################                    953us
.cfg               378B | ######################                   1.3ms
.pro               384B | #######################                  1.7ms
.json              399B | #######################                  1.7ms
.j2                417B | #######################                  1.6ms
.toml              560B | #########################                3.2ms
.tf                610B | ########################                 2.4ms
.mod               634B | #########################                3.6ms
.yml               679B | ##########################               4.5ms
.html              683B | ###########################              6.3ms
.sh                763B | ##########################               4.6ms
.kts               817B | ##########################               5.1ms
.xml               918B | ###########################              6.2ms
.yaml              928B | ##########################               5.1ms
.svg              1.0KB | ############################             8.8ms
.ts               1.1KB | ##############################           18.6ms
.d                1.2KB | ############################             8.4ms
.hcl              1.3KB | #############################            12.5ms
.rego             1.3KB | ############################             8.5ms
.js               1.7KB | ###############################          20.0ms
.rs               2.5KB | ###############################          19.9ms
.mjs              2.8KB | ##################################       52.8ms
.md               2.9KB | ##################################       53.6ms
.kt               3.5KB | #################################        38.8ms
.lock             3.8KB | ###############################          23.4ms
.go               4.1KB | ###################################      91.0ms
.css              4.1KB | ####################################     99.4ms
.sum              4.1KB | ##################################       62.3ms
.vue              4.6KB | ####################################     117.3ms
.txt              4.8KB | ##################################       56.6ms
.java            16.5KB | ######################################## 343.6ms

Avg Compression Ratio vs Avg File Size (sorted by size)
================================================================================
.timestamp          48B | ####                                     10.4%
.list               62B | ####                                     11.6%
.dockerignore       71B | #######                                  19.7%
.pub               146B | #####                                    13.0%
.properties        156B | ##########                               26.0%
.tag               177B | #######                                  19.8%
.gitignore         182B | ########                                 21.7%
.conf              213B | #########                                24.6%
.editorconfig      216B | ########                                 21.8%
.backend           229B | ##########                               27.1%
.example           276B | #######                                  19.2%
.service           334B | ########                                 21.8%
.cfg               378B | #########                                24.3%
.pro               384B | ############                             32.3%
.json              399B | ############                             30.6%
.j2                417B | ################                         40.2%
.toml              560B | #############                            33.2%
.tf                610B | ###############                          38.9%
.mod               634B | ############                             30.0%
.yml               679B | ##############                           35.7%
.html              683B | ###########                              27.8%
.sh                763B | ##############                           35.9%
.kts               817B | ################                         40.3%
.xml               918B | #################                        43.0%
.yaml              928B | ######################                   57.1%
.svg              1.0KB | #############                            33.8%
.ts               1.1KB | ##############                           37.3%
.d                1.2KB | #############################            74.1%
.hcl              1.3KB | #########                                23.4%
.rego             1.3KB | #######################                  58.1%
.js               1.7KB | #################                        44.3%
.rs               2.5KB | ####################                     51.3%
.mjs              2.8KB | ##################                       46.3%
.md               2.9KB | #############                            34.3%
.kt               3.5KB | ######################                   56.3%
.lock             3.8KB | ########################                 61.2%
.go               4.1KB | ###################                      48.5%
.css              4.1KB | ###################                      47.6%
.sum              4.1KB | ##############                           36.3%
.vue              4.6KB | ################                         40.7%
.txt              4.8KB | ##########                               26.7%
.java            16.5KB | ##############################           75.0%
```

Overall: **39.8% average compression** across 1112 files. Compression time
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
