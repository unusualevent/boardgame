package boardgame

import (
	"bytes"
	"io"
	"testing"
)

func TestWriterReader(t *testing.T) {
	cases := []string{
		"hello hello hello",
		"func main() {\n\tfmt.Println(`hello`)\n}",
		"aaa bbb aaa bbb aaa bbb",
	}
	for _, tc := range cases {
		var buf bytes.Buffer
		w := NewWriter(&buf)
		_, err := w.Write([]byte(tc))
		if err != nil {
			t.Fatalf("Write(%q): %v", tc, err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("Close(%q): %v", tc, err)
		}

		r := NewReader(&buf)
		got, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("ReadAll(%q): %v", tc, err)
		}
		if !bytes.Equal(got, []byte(tc)) {
			t.Errorf("stream round-trip failed for %q: got %q", tc, got)
		}
	}
}

func TestReaderSmallBuffer(t *testing.T) {
	src := []byte("repeated text repeated text repeated text")
	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.Write(src)
	w.Close()

	r := NewReader(&buf)
	var result []byte
	small := make([]byte, 5)
	for {
		n, err := r.Read(small)
		result = append(result, small[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(result, src) {
		t.Errorf("small-buffer read failed: got %q", result)
	}
}

func TestReaderEmpty(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.Close()

	r := NewReader(&buf)
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %q", got)
	}
}
