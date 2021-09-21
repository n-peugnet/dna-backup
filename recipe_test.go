package main

import "testing"

func TestRecipe(t *testing.T) {
	c1 := &StoredChunk{Id: &ChunkId{0, 1}}
	c2 := &StoredChunk{Id: &ChunkId{0, 2}}
	c3 := &StoredChunk{Id: &ChunkId{0, 3}}
	c4 := &StoredChunk{Id: &ChunkId{0, 4}}
	c5 := &StoredChunk{Id: &ChunkId{0, 5}}
	c6 := &StoredChunk{Id: &ChunkId{0, 6}}
	c7 := &StoredChunk{Id: &ChunkId{0, 7}}
	source := Recipe{c1, c2, c3, c4}
	target := Recipe{c2, c5, c3, c6, c4, c7}
	patch := diffRecipe(source, target)
	assertSame(t, []RecipeDel{0}, patch.Del, "Patch del part")
	assertSame(t, []RecipeIns{
		{1, []Chunk{c5}},
		{3, []Chunk{c6}},
		{5, []Chunk{c7}},
	}, patch.Ins, "Patch ins part")
	actual := patchRecipe(source, patch)
	assertSame(t, target, actual, "Target obtained from patch application")
}
