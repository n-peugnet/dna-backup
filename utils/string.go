package utils

import (
	"fmt"
	"strings"
)

func Unprefix(path string, prefix string) (string, error) {
	if !strings.HasPrefix(path, prefix) {
		return path, fmt.Errorf("%q is not prefixed by %q", path, prefix)
	} else {
		return path[len(prefix):], nil
	}
}
