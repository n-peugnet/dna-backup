package utils

import (
	"compress/zlib"
	"io"
)

// NopCloser returns a WriteCloser with a no-op Close method wrapping
// the provided Writer w.
func NopCloser(w io.Writer) io.WriteCloser {
	return nopCloser{w}
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

// ZlibReader wraps a reader with a new zlib.Reader.
func ZlibReader(r io.Reader) (io.ReadCloser, error) {
	return zlib.NewReader(r)
}

// ZlibWrier wraps a writer with a new zlib.Writer.
func ZlibWriter(w io.Writer) io.WriteCloser {
	return zlib.NewWriter(w)
}
