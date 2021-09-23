package slice_test

import (
	"testing"

	"github.com/n-peugnet/dna-backup/slice"
	"github.com/n-peugnet/dna-backup/testutils"
)

func TestPatch(t *testing.T) {
	source := slice.Slice{1, 2, 3, 4}
	target := slice.Slice{2, 5, 3, 6, 4, 7, 8}
	patch := slice.Diff(source, target)
	testutils.AssertSame(t, []slice.Del{0}, patch.Del, "Patch del part")
	testutils.AssertSame(t, []slice.Ins{
		{1, slice.Slice{5}},
		{3, slice.Slice{6}},
		{5, slice.Slice{7, 8}},
	}, patch.Ins, "Patch ins part")
	actual := slice.Patch(source, patch)
	testutils.AssertSame(t, target, actual, "Target obtained from patch application")
}

func TestEmptyPatch(t *testing.T) {
	source := slice.Slice{1, 2, 3, 4}
	target := slice.Slice{1, 2, 3, 4}
	patch := slice.Diff(source, target)
	testutils.AssertSame(t, *new([]slice.Del), patch.Del, "Patch del part")
	testutils.AssertSame(t, *new([]slice.Ins), patch.Ins, "Patch ins part")
	actual := slice.Patch(source, patch)
	testutils.AssertSame(t, target, actual, "Target obtained from patch application")
}

type i struct {
	int
}

func TestStruct(t *testing.T) {
	c1, c2, c3, c4, c5, c6, c7, c8 := &i{1}, &i{2}, &i{3}, &i{4}, &i{5}, &i{6}, &i{7}, &i{8}
	source := slice.Slice{c1, c2, c3, c4}
	target := slice.Slice{&i{5}, c2, c5, c6, &i{4}, c7, &i{8}}
	patch := slice.Diff(source, target)
	testutils.AssertSame(t, []slice.Del{0, 2}, patch.Del, "Patch del part")
	testutils.AssertSame(t, []slice.Ins{
		{0, slice.Slice{c5}},
		{2, slice.Slice{c5, c6}},
		{5, slice.Slice{c7, c8}},
	}, patch.Ins, "Patch ins part")
	actual := slice.Patch(source, patch)
	testutils.AssertSame(t, target, actual, "Target obtained from patch application")
}

func TestDeltaString(t *testing.T) {
	delta := slice.Delta{
		Del: []slice.Del{0, 3, 4},
		Ins: []slice.Ins{
			{2, slice.Slice{6, 7, 8}},
			{5, slice.Slice{5}},
		},
	}
	testutils.AssertSame(
		t,
		"{Del: [0 3 4] Ins: [{idx:2 count:3} {idx:5 count:1}]}",
		delta.String(),
		"Delta string representation",
	)
}
