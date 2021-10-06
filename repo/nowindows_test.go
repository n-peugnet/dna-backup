// +build !windows

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
