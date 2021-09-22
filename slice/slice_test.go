package slice

import (
	"testing"

	"github.com/n-peugnet/dna-backup/testutils"
)

func TestPatch(t *testing.T) {
	source := Slice{1, 2, 3, 4}
	target := Slice{2, 5, 3, 6, 4, 7, 8}
	patch := Diff(source, target)
	testutils.AssertSame(t, []Del{0}, patch.Del, "Patch del part")
	testutils.AssertSame(t, []Ins{
		{1, Slice{5}},
		{3, Slice{6}},
		{5, Slice{7, 8}},
	}, patch.Ins, "Patch ins part")
	actual := Patch(source, patch)
	testutils.AssertSame(t, target, actual, "Target obtained from patch application")
}

func TestEmptyPatch(t *testing.T) {
	source := Slice{1, 2, 3, 4}
	target := Slice{1, 2, 3, 4}
	patch := Diff(source, target)
	testutils.AssertSame(t, *new([]Del), patch.Del, "Patch del part")
	testutils.AssertSame(t, *new([]Ins), patch.Ins, "Patch ins part")
	actual := Patch(source, patch)
	testutils.AssertSame(t, target, actual, "Target obtained from patch application")
}
