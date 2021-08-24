package main

import "testing"

func TestIsStored(t *testing.T) {
	stored := Chunk{Id: &ChunkId{0, 0}}
	if !stored.isStored() {
		t.Error("Chunk ", stored, " should be stored")
	}
	unstored := Chunk{}
	if unstored.isStored() {
		t.Error("Chunk ", unstored, " should not be stored")
	}
}
