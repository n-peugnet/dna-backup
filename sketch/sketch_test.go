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

package sketch

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/chmduquesne/rollinghash/rabinkarp64"
)

func TestSketchChunk(t *testing.T) {
	var sketch, expected Sketch
	var err error
	dataDir := "testdata"
	pol, err := rabinkarp64.RandomPolynomial(1)
	if err != nil {
		t.Fatal(err)
	}

	c0, err := os.Open(filepath.Join(dataDir, "000000000000000"))
	if err != nil {
		t.Fatal(err)
	}
	sketch, err = SketchChunk(c0, pol, 8<<10, 32, 3, 4)
	if err != nil {
		t.Fatal(err)
	}
	expected = Sketch{429857165471867, 6595034117354675, 8697818304802825}
	if !reflect.DeepEqual(sketch, expected) {
		t.Errorf("Sketch does not match, expected: %d, actual: %d", expected, sketch)
	}

	c14, err := os.Open(filepath.Join(dataDir, "000000000000014"))
	sketch, err = SketchChunk(c14, pol, 8<<10, 32, 3, 4)
	if err != nil {
		t.Error(err)
	}
	expected = Sketch{658454504014104}
	if !reflect.DeepEqual(sketch, expected) {
		t.Errorf("Sketch does not match, expected: %d, actual: %d", expected, sketch)
	}
}
