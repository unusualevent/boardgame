# Boardgame Wire Format

Boardgame compresses source code in three stages: RLE preprocessing,
table substitution, and 7-bit packing. Non-ASCII bytes (UTF-8, etc.)
are DEL-escaped before compression and pass through transparently.

## Byte ranges

| Range       | Meaning                          |
|-------------|----------------------------------|
| `0x00`      | Table entry delimiter / command  |
| `0x01–0x08` | Direct table reference (1 byte)  |
| `0x09`      | Literal tab                      |
| `0x0A`      | Literal newline                  |
| `0x0B–0x1D` | Direct table reference (1 byte)  |
| `0x1E`      | RLE tab marker                   |
| `0x1F`      | RLE space marker                 |
| `0x20–0x7E` | Literal ASCII glyphs             |
| `0x7F`      | DEL — escape byte                |
| `0x80–0xFF` | Only inside DEL-escaped values   |

## Stage 1: RLE preprocessing

Runs of 4–15 consecutive spaces or tabs are collapsed into 2-byte
RLE tokens before table substitution. This shrinks the input to the
suffix array and frees table slots for higher-value patterns.

```
<marker> <count_byte>
```

- `0x1F` = space run marker, `0x1E` = tab run marker
- Count byte = `run_length - 4 + 0x0B`, mapping lengths 4–15 to bytes
  `0x0B–0x16` (all non-literal, acting as barriers in the suffix array)
- Runs longer than 15 are split into multiple RLE pairs plus remaining
  literal characters

### Expansion

The decoder restores RLE tokens after table expansion: each
`{marker, count_byte}` pair is replaced with the corresponding run of
spaces or tabs.

## Stage 2: Table substitution

The intermediate stream is a sequence of commands and literals that the
encoder produces and the decoder consumes.

### Define a table entry

```
0x00 <sequence bytes> 0x00
```

The sequence is assigned to the **lowest free slot** (1–255). Definitions
may appear anywhere in the stream, not only at the beginning.

### Direct reference (slots 0x01–0x1D)

A single byte in the range `0x01–0x1D` expands to the sequence stored in
that slot. Bytes `0x09` (tab), `0x0A` (newline), `0x1E` (RLE tab), and
`0x1F` (RLE space) are reserved and cannot be used as slot IDs.

### Extended reference (slots 0x1E–0xFF)

```
0x00 0x7F <slot>
```

The 3-byte sequence null–DEL–byte references any slot by its full 8-bit
ID. This allows up to 251 table entries total (4 byte values — tab,
newline, and the two RLE markers — are reserved and cannot be slots).

### Free a slot (slots 0x01–0x1D)

```
0x00 <slot>
```

An unpaired null followed by a direct-range byte (`0x01–0x1D`) frees
that slot **if it is currently occupied**. The freed slot becomes
available for future definitions. If the slot is not occupied, the bytes
are interpreted as a table entry definition instead.

### DEL escape (literal 8-bit value)

```
0x7F <byte>
```

A standalone DEL byte (not preceded by null) passes the next byte through
unchanged, allowing arbitrary 8-bit values in the output.

## Stage 3: 7-bit packing

All bytes in the intermediate stream have bit 7 = 0 (they are ≤ 0x7F),
so each can be represented in 7 bits. The packer writes each byte as 7
bits, MSB first, concatenating them into a bitstream.

When the packer encounters `0x7F` (DEL), it writes the 7-bit DEL value
followed by the **next byte as 8 raw bits** (since DEL-escaped values
may have bit 7 set).

### Packed output layout

```
[padding_byte] [packed_data...]
```

- **Byte 0** — number of padding bits (0–6) appended to the final byte
  of packed data to reach a byte boundary.
- **Bytes 1..n** — the packed bitstream.

### Unpacking

Read 7 bits at a time. If the value is `0x7F`, read the next 8 bits as a
raw byte and emit both the DEL marker and the raw byte. Otherwise emit
the 7-bit value directly. Stop when fewer than 7 usable bits remain
(accounting for the padding count).
