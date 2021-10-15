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

package testutils

import (
	"os"
	"reflect"
	"testing"
)

func AssertSame(t *testing.T, expected interface{}, actual interface{}, prefix string) {
	if !reflect.DeepEqual(expected, actual) {
		t.Error(prefix, "do not match, expected:", expected, ", actual:", actual)
	}
}

func AssertSameFile(t *testing.T, expected string, actual string, prefix string) {
	efContent, err := os.ReadFile(expected)
	if err != nil {
		t.Fatalf("%s Error reading from expected file '%s': %s", prefix, expected, err)
	}
	afContent, err := os.ReadFile(actual)
	if err != nil {
		t.Fatalf("%s Error reading from expected file '%s': %s", prefix, actual, err)
	}
	AssertSame(t, efContent, afContent, prefix+" files")
}

func AssertLen(t *testing.T, expected int, actual interface{}, prefix string) {
	s := reflect.ValueOf(actual)
	if s.Len() != expected {
		t.Fatal(prefix, "incorrect length, expected:", expected, ", actual:", s.Len())
	}
}
