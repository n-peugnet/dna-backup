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

type i struct {
	int
}

func TestStruct(t *testing.T) {
	c1, c2, c3, c4, c5, c6, c7, c8 := &i{1}, &i{2}, &i{3}, &i{4}, &i{5}, &i{6}, &i{7}, &i{8}
	source := Slice{c1, c2, c3, c4}
	target := Slice{&i{5}, c2, c5, c6, &i{4}, c7, &i{8}}
	patch := Diff(source, target)
	testutils.AssertSame(t, []Del{0, 2}, patch.Del, "Patch del part")
	testutils.AssertSame(t, []Ins{
		{0, Slice{c5}},
		{2, Slice{c5, c6}},
		{5, Slice{c7, c8}},
	}, patch.Ins, "Patch ins part")
	actual := Patch(source, patch)
	testutils.AssertSame(t, target, actual, "Target obtained from patch application")
}
