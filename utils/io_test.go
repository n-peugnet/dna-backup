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

package utils_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/n-peugnet/dna-backup/utils"
)

func TestNopCloser(t *testing.T) {
	var buff bytes.Buffer
	w := utils.NopCloser(&buff)
	w.Close()
	n, err := buff.WriteString("test")
	if err != nil {
		t.Error(err)
	}
	if n != 4 {
		t.Error("expected: 4, actual:", n)
	}
}

type wrapper struct {
	n string
	r utils.ReadWrapper
	w utils.WriteWrapper
}

func TestWrappers(t *testing.T) {
	wrappers := []wrapper{
		{"Zlib", utils.ZlibReader, utils.ZlibWriter},
		{"Nop", utils.NopReadWrapper, utils.NopWriteWrapper},
	}
	for _, wrapper := range wrappers {
		t.Run(wrapper.n, func(t *testing.T) {
			testWrapper(t, wrapper)
		})
	}
}

func testWrapper(t *testing.T, wrapper wrapper) {
	var buff bytes.Buffer
	var err error
	w := wrapper.w(&buff)
	n, err := w.Write([]byte("test"))
	if err != nil {
		t.Error(wrapper.n, err)
	}
	if n != 4 {
		t.Error(wrapper.n, "expected: 4, actual:", n)
	}
	err = w.Close()
	if err != nil {
		t.Error(wrapper.n, err)
	}
	r, err := wrapper.r(&buff)
	if err != nil {
		t.Error(wrapper.n, err)
	}
	b := make([]byte, 4)
	n, err = r.Read(b)
	if n != 4 {
		t.Error(wrapper.n, "expected: 4, actual:", n)
	}
	if err != nil && err != io.EOF {
		t.Error(wrapper.n, err)
	}
	if !bytes.Equal(b, []byte("test")) {
		t.Errorf("%s, expected: %q, actual: %q", "test", wrapper.n, b)
	}
	n, err = r.Read(b)
	if err != io.EOF {
		t.Error(wrapper.n, err)
	}
}
