package boardgame

// pack7 packs a byte slice into a 7-bit stream. The first byte of the
// output stores the number of padding bits (0–6) in the final byte.
// Each input byte with bit 7 = 0 is written as 7 bits. DEL (0x7F)
// in the input signals that the next byte is written as 8 raw bits.
func pack7(src []byte) []byte {
	var w bitWriter
	for i := 0; i < len(src); i++ {
		b := src[i]
		if b == delByte {
			w.writeBits(delByte, 7)
			i++
			if i < len(src) {
				w.writeBits(uint(src[i]), 8)
			}
		} else {
			w.writeBits(uint(b), 7)
		}
	}
	packed := w.bytes()
	totalBits := w.totalBits
	var padding byte
	if totalBits%8 != 0 {
		padding = byte(8 - totalBits%8)
	}
	return append([]byte{padding}, packed...)
}

// unpack7 unpacks a 7-bit stream back to bytes. The first byte indicates
// how many padding bits trail the data. DEL (0x7F) signals that the next
// 8 bits are a raw byte value; both the DEL marker and the raw byte are
// emitted so the table layer can see them.
func unpack7(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}
	padding := int(src[0])
	data := src[1:]
	usableBits := len(data)*8 - padding
	r := bitReader{data: data}
	var out []byte
	for int(r.pos)+7 <= usableBits {
		v, ok := r.readBits(7)
		if !ok {
			break
		}
		if v == delByte {
			if int(r.pos)+8 > usableBits {
				return nil, ErrTruncated
			}
			raw, ok := r.readBits(8)
			if !ok {
				return nil, ErrTruncated
			}
			out = append(out, delByte, byte(raw))
		} else {
			out = append(out, byte(v))
		}
	}
	return out, nil
}

// bitWriter accumulates bits into a byte slice, MSB first.
type bitWriter struct {
	buf       []byte
	n         uint // bits written into current byte
	totalBits int
}

func (w *bitWriter) writeBits(val uint, nbits int) {
	w.totalBits += nbits
	for i := nbits - 1; i >= 0; i-- {
		if w.n == 0 {
			w.buf = append(w.buf, 0)
		}
		if val&(1<<uint(i)) != 0 {
			w.buf[len(w.buf)-1] |= 1 << (7 - w.n)
		}
		w.n++
		if w.n == 8 {
			w.n = 0
		}
	}
}

func (w *bitWriter) bytes() []byte {
	return w.buf
}

// bitReader reads bits from a byte slice, MSB first.
type bitReader struct {
	data []byte
	pos  uint // bit position
}

func (r *bitReader) readBits(n int) (uint, bool) {
	if r.remaining() < n {
		return 0, false
	}
	var val uint
	for i := 0; i < n; i++ {
		byteIdx := r.pos / 8
		bitIdx := 7 - (r.pos % 8)
		if r.data[byteIdx]&(1<<bitIdx) != 0 {
			val |= 1 << uint(n-1-i)
		}
		r.pos++
	}
	return val, true
}

func (r *bitReader) remaining() int {
	total := len(r.data) * 8
	return total - int(r.pos)
}
