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

package cache

import (
	"bytes"
	"testing"
)

func TestFifoChunkCache(t *testing.T) {
	var v []byte
	var e bool
	var cache Cacher = NewFifoCache(3)
	k0 := 0
	k1 := 1
	k2 := 2
	k3 := 3
	v0 := []byte{'0'}
	v1 := []byte{'1'}
	v2 := []byte{'2'}
	v3 := []byte{'3'}

	if cache.Len() != 0 {
		t.Fatal("Cache should be of size 0")
	}

	v, e = cache.Get(k0)
	if e {
		t.Fatal("There should not be any value")
	}

	cache.Set(k0, v0)
	cache.Set(k1, v1)
	cache.Set(k2, v2)

	if cache.Len() != 3 {
		t.Fatal("Cache should be of size 3")
	}

	v, e = cache.Get(k0)
	if !e {
		t.Fatal("Value should exist for k0")
	}
	if bytes.Compare(v, v0) != 0 {
		t.Fatal("Value for k0 does not match")
	}

	cache.Set(k3, v3)

	if cache.Len() != 3 {
		t.Fatal("Cache should still be of size 3")
	}

	v, e = cache.Get(k0)
	if e {
		t.Fatal("Value should not exist for k0")
	}

	v, e = cache.Get(k1)
	if !e {
		t.Fatal("Value should exist for k1")
	}
	if bytes.Compare(v, v1) != 0 {
		t.Fatal("Value for k1 does not match")
	}

	v, e = cache.Get(k2)
	if !e {
		t.Fatal("Value should exist for k2")
	}
	if bytes.Compare(v, v2) != 0 {
		t.Fatal("Value for k2 does not match")
	}

	v, e = cache.Get(k3)
	if !e {
		t.Fatal("Value should exist for k3")
	}
	if bytes.Compare(v, v3) != 0 {
		t.Fatal("Value for k3 does not match")
	}
}
