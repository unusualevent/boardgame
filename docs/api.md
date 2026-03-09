# API Reference

```
import "git.risottobias.org/claude/boardgame"
```

## Functions

### Encode

```go
func Encode(src []byte) ([]byte, error)
```

Compresses `src` using table substitution and 7-bit packing. Printable
ASCII (`0x20–0x7E`), tab (`0x09`), and newline (`0x0A`) participate in
compression. Non-ASCII bytes (UTF-8, etc.) are DEL-escaped and pass
through transparently — they act as barriers in the candidate search
but round-trip correctly.

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

### NewWriter

```go
func NewWriter(w io.Writer) *Writer
```

Returns a writer that buffers data and compresses it on `Close`.
Implements `io.WriteCloser`.

### NewReader

```go
func NewReader(r io.Reader) *Reader
```

Returns a reader that decompresses boardgame-encoded data from `r`.
The entire compressed input is read and decompressed on the first
`Read` call. Implements `io.Reader`.

## Errors

| Error                | Meaning                                              |
|----------------------|------------------------------------------------------|
| `ErrTooManyEntries`  | More than 255 table entries defined                  |
| `ErrUnterminatedSeq` | Table entry missing its closing `0x00`               |
| `ErrBadRef`          | Reference to an undefined or out-of-range table slot |
| `ErrByteOutOfRange`  | (No longer returned by Encode; kept for compatibility) |
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
