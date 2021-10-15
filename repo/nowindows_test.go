// +build !windows

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

package repo

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/n-peugnet/dna-backup/logger"
	"github.com/n-peugnet/dna-backup/testutils"
	"github.com/n-peugnet/dna-backup/utils"
)

func TestNotReadable(t *testing.T) {
	var output bytes.Buffer
	logger.SetOutput(&output)
	defer logger.SetOutput(os.Stderr)
	tmpDir := t.TempDir()
	f, err := os.OpenFile(filepath.Join(tmpDir, "notreadable"), os.O_CREATE, 0000)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	var buff bytes.Buffer
	files := listFiles(tmpDir)
	testutils.AssertLen(t, 1, files, "Files")
	concatFiles(&files, utils.NopCloser(&buff))
	testutils.AssertLen(t, 0, files, "Files")
	testutils.AssertLen(t, 0, buff.Bytes(), "Buffer")
	if !strings.Contains(output.String(), "notreadable") {
		t.Errorf("log should contain a warning for notreadable, actual %q", &output)
	}
}
