package slice

import (
	"testing"

	"github.com/n-peugnet/dna-backup/testutils"
)

func TestPatch(t *testing.T) {
	source := Slice{1, 2, 3, 4}
	target := Slice{2, 5, 3, 6, 4, 7, 8}
	patch := DiffSlice(source, target)
	testutils.AssertSame(t, []SliceDel{0}, patch.Del, "Patch del part")
	testutils.AssertSame(t, []SliceIns{
		{1, Slice{5}},
		{3, Slice{6}},
		{5, Slice{7, 8}},
	}, patch.Ins, "Patch ins part")
	actual := PatchSlice(source, patch)
	testutils.AssertSame(t, target, actual, "Target obtained from patch application")
}
