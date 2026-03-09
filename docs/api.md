# API Reference

```
import "boardgame"
```

## Functions

### Encode

```go
func Encode(src []byte) ([]byte, error)
```

Compresses `src` using table substitution and 7-bit packing. All input
bytes must be in the glyph range `0x20–0x73`; returns `ErrByteOutOfRange`
otherwise.

### Decode

```go
func Decode(src []byte) ([]byte, error)
```

Decompresses a boardgame-encoded byte stream back to the original input.

### Stats

```go
func Stats(original, compressed []byte) (origLen, compLen int, ratio float64)
```

Returns the original length, compressed length, and compression ratio
(0.0 = no savings, 1.0 = fully eliminated). Useful for diagnostics.

## Errors

| Error                | Meaning                                              |
|----------------------|------------------------------------------------------|
| `ErrTooManyEntries`  | More than 255 table entries defined                  |
| `ErrUnterminatedSeq` | Table entry missing its closing `0x00`               |
| `ErrBadRef`          | Reference to an undefined or out-of-range table slot |
| `ErrByteOutOfRange`  | Input byte outside the valid glyph range `0x20–0x73` |
| `ErrTruncated`       | Bitstream ended mid-value (e.g. after a DEL escape)  |
| `ErrNoFreeSlot`      | All 255 table slots are occupied                     |

## Usage example

```go
original := []byte("hello hello hello")

compressed, err := boardgame.Encode(original)
if err != nil {
    log.Fatal(err)
}

restored, err := boardgame.Decode(compressed)
if err != nil {
    log.Fatal(err)
}
// restored == original

_, compLen, ratio := boardgame.Stats(original, compressed)
fmt.Printf("compressed to %d bytes (%.1f%% savings)\n", compLen, ratio*100)
```
