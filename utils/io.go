/* Copyright (C) 2021 Nicolas Peugnet <n.peugnet@free.fr>

   This file is part of dna-backup.

   dna-backup is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   dna-backup is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with dna-backup.  If not, see <https://www.gnu.org/licenses/>. */

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

type ReadWrapper func(r io.Reader) (io.ReadCloser, error)
type WriteWrapper func(w io.Writer) io.WriteCloser

// ZlibReader wraps a reader with a new zlib.Reader.
func ZlibReader(r io.Reader) (io.ReadCloser, error) {
	return zlib.NewReader(r)
}

// ZlibWrier wraps a writer with a new zlib.Writer.
func ZlibWriter(w io.Writer) io.WriteCloser {
	return zlib.NewWriter(w)
}

func NopReadWrapper(r io.Reader) (io.ReadCloser, error) {
	return io.NopCloser(r), nil
}

func NopWriteWrapper(w io.Writer) io.WriteCloser {
	return NopCloser(w)
}

type WriteCounter struct {
	w     io.Writer
	count int
}

func NewWriteCounter(writer io.Writer) *WriteCounter {
	return &WriteCounter{w: writer}
}

func (c *WriteCounter) Write(p []byte) (n int, err error) {
	n, err = c.w.Write(p)
	c.count += n
	return
}

func (c *WriteCounter) Count() int {
	return c.count
}
