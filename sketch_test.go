package main

import (
	"path"
	"reflect"
	"testing"
)

func TestSketchChunk(t *testing.T) {
	dataDir := path.Join("test", "data", "repo_8k")
	chunks := make(chan IdentifiedChunk, 16)
	repo := NewRepo(dataDir)
	versions := repo.loadVersions()
	go repo.loadChunks(versions, chunks)
	var i int
	for c := range chunks {
		if i < 1 {
			sketch, err := SketchChunk(c, repo.pol, 8<<10, 32, 3, 4)
			if err != nil {
				t.Error(err)
			}
			expected := Sketch{429857165471867, 6595034117354675, 8697818304802825}
			if !reflect.DeepEqual(sketch, expected) {
				t.Errorf("Sketch does not match, expected: %d, actual: %d", expected, sketch)
			}
		}
		if i == 14 {
			sketch, err := SketchChunk(c, repo.pol, 8<<10, 32, 3, 4)
			if err != nil {
				t.Error(err)
			}
			expected := Sketch{658454504014104}
			if !reflect.DeepEqual(sketch, expected) {
				t.Errorf("Sketch does not match, expected: %d, actual: %d", expected, sketch)
			}
		}
		i++
	}
}
