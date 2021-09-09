package utils

import (
	"path/filepath"
	"strings"
)

func TrimTrailingSeparator(path string) string {
	return strings.TrimSuffix(path, string(filepath.Separator))
}
