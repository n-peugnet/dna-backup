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
	"testing"

	"github.com/n-peugnet/dna-backup/utils"
)

func TestUnprefix(t *testing.T) {
	str, err := utils.Unprefix("foo/bar", "foo/")
	if str != "bar" {
		t.Errorf("expected: %q, actual: %q", "bar", str)
	}
	if err != nil {
		t.Error(err)
	}
	str, err = utils.Unprefix("foo/bar", "baz")
	if str != "foo/bar" {
		t.Errorf("expected: %q, actual: %q", "foo/bar", str)
	}
	if err == nil {
		t.Error("err should not be nil")
	}
}
