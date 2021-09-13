package sketch

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/chmduquesne/rollinghash/rabinkarp64"
)

func TestSketchChunk(t *testing.T) {
	var sketch, expected Sketch
	var err error
	dataDir := "testdata"
	pol, err := rabinkarp64.RandomPolynomial(1)
	if err != nil {
		t.Fatal(err)
	}

	c0, err := os.Open(filepath.Join(dataDir, "000000000000000"))
	if err != nil {
		t.Fatal(err)
	}
	sketch, err = SketchChunk(c0, pol, 8<<10, 32, 3, 4)
	if err != nil {
		t.Fatal(err)
	}
	expected = Sketch{429857165471867, 6595034117354675, 8697818304802825}
	if !reflect.DeepEqual(sketch, expected) {
		t.Errorf("Sketch does not match, expected: %d, actual: %d", expected, sketch)
	}

	c14, err := os.Open(filepath.Join(dataDir, "000000000000014"))
	sketch, err = SketchChunk(c14, pol, 8<<10, 32, 3, 4)
	if err != nil {
		t.Error(err)
	}
	expected = Sketch{658454504014104}
	if !reflect.DeepEqual(sketch, expected) {
		t.Errorf("Sketch does not match, expected: %d, actual: %d", expected, sketch)
	}
}
