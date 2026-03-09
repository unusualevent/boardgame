package boardgame

import "io"

// NewWriter returns a writer that compresses data written to it.
// All data is buffered until Close is called, at which point the
// compressed output is written to the underlying writer.
//
// Callers must call Close to flush the compressed data.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Writer compresses data written to it using boardgame encoding.
// The compressed output is flushed to the underlying writer on Close.
type Writer struct {
	w   io.Writer
	buf []byte
}

// Write appends p to the internal buffer. Compression happens on Close.
func (w *Writer) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	return len(p), nil
}

// Close compresses the buffered data and writes it to the underlying writer.
func (w *Writer) Close() error {
	enc, err := Encode(w.buf)
	if err != nil {
		return err
	}
	_, err = w.w.Write(enc)
	w.buf = nil
	return err
}

// NewReader returns a reader that decompresses boardgame-encoded data.
// The entire compressed input is read from r on the first Read call
// and decompressed.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

// Reader decompresses boardgame-encoded data from an underlying reader.
type Reader struct {
	r   io.Reader
	buf []byte
	pos int
	err error
}

// Read decompresses data into p. On the first call, the entire compressed
// input is read from the underlying reader and decompressed.
func (r *Reader) Read(p []byte) (int, error) {
	if r.buf == nil && r.err == nil {
		compressed, err := io.ReadAll(r.r)
		if err != nil {
			r.err = err
			return 0, err
		}
		r.buf, r.err = Decode(compressed)
		if r.err != nil {
			return 0, r.err
		}
	}
	if r.err != nil && r.err != io.EOF {
		return 0, r.err
	}
	if r.pos >= len(r.buf) {
		return 0, io.EOF
	}
	n := copy(p, r.buf[r.pos:])
	r.pos += n
	if r.pos >= len(r.buf) {
		return n, io.EOF
	}
	return n, nil
}
