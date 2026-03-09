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

Input must be printable ASCII (`0x20`–`0x7E`) plus tab (`0x09`) and newline
(`0x0A`). See [docs/api.md](docs/api.md) for full details.

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

## Testing

```
go test ./...                          # unit tests
go test -fuzz=FuzzRoundTrip -fuzztime=30s  # fuzz: encode/decode round-trip
go test -fuzz=FuzzDecode -fuzztime=30s     # fuzz: decode crash resistance
```
