# Boardgame Wire Format

Boardgame compresses ASCII source code in two stages: table substitution
followed by 7-bit packing.

## Byte ranges

| Range       | Meaning                          |
|-------------|----------------------------------|
| `0x00`      | Table entry delimiter / command  |
| `0x01–0x08` | Direct table reference (1 byte)  |
| `0x09`      | Literal tab                      |
| `0x0A`      | Literal newline                  |
| `0x0B–0x19` | Direct table reference (1 byte)  |
| `0x1A–0x1F` | Reserved (extended ref range)    |
| `0x20–0x7E` | Literal ASCII glyphs             |
| `0x7F`      | DEL — escape byte                |
| `0x80–0xFF` | Only inside DEL-escaped values   |

## Stage 1: Table substitution

The intermediate stream is a sequence of commands and literals that the
encoder produces and the decoder consumes.

### Define a table entry

```
0x00 <sequence bytes> 0x00
```

The sequence is assigned to the **lowest free slot** (1–255). Definitions
may appear anywhere in the stream, not only at the beginning.

### Direct reference (slots 0x01–0x19)

A single byte in the range `0x01–0x19` expands to the sequence stored in
that slot.

### Extended reference (slots 0x1A–0xFF)

```
0x00 0x7F <slot>
```

The 3-byte sequence null–DEL–byte references any slot by its full 8-bit
ID. This allows up to 255 table entries total.

### Free a slot (slots 0x01–0x19)

```
0x00 <slot>
```

An unpaired null followed by a direct-range byte (`0x01–0x19`) frees
that slot **if it is currently occupied**. The freed slot becomes
available for future definitions. If the slot is not occupied, the bytes
are interpreted as a table entry definition instead.

### DEL escape (literal 8-bit value)

```
0x7F <byte>
```

A standalone DEL byte (not preceded by null) passes the next byte through
unchanged, allowing arbitrary 8-bit values in the output.

## Stage 2: 7-bit packing

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
