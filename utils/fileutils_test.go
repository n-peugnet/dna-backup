package utils

import (
	"path/filepath"
	"testing"
)

func TestTrimTrailingSeparator(t *testing.T) {
	if TrimTrailingSeparator("test"+string(filepath.Separator)) != "test" {
		t.Error("Seprator should have been trimmed")
	}
	if TrimTrailingSeparator("test") != "test" {
		t.Error("Path should not have changed")
	}
}
