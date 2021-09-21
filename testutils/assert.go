package testutils

import (
	"os"
	"reflect"
	"testing"
)

func AssertSame(t *testing.T, expected interface{}, actual interface{}, prefix string) {
	if !reflect.DeepEqual(expected, actual) {
		t.Error(prefix, "do not match, expected:", expected, ", actual:", actual)
	}
}

func AssertSameFile(t *testing.T, expected string, actual string, prefix string) {
	efContent, err := os.ReadFile(expected)
	if err != nil {
		t.Fatalf("%s Error reading from expected file '%s': %s", prefix, expected, err)
	}
	afContent, err := os.ReadFile(actual)
	if err != nil {
		t.Fatalf("%s Error reading from expected file '%s': %s", prefix, actual, err)
	}
	AssertSame(t, efContent, afContent, prefix+" files")
}

func AssertLen(t *testing.T, expected int, actual interface{}, prefix string) {
	s := reflect.ValueOf(actual)
	if s.Len() != expected {
		t.Fatal(prefix, "incorrect length, expected:", expected, ", actual:", s.Len())
	}
}
