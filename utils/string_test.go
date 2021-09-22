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
