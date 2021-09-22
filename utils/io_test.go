package utils_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/n-peugnet/dna-backup/utils"
)

func TestNopCloser(t *testing.T) {
	var buff bytes.Buffer
	w := utils.NopCloser(&buff)
	w.Close()
	n, err := buff.WriteString("test")
	if err != nil {
		t.Error(err)
	}
	if n != 4 {
		t.Error("expected: 4, actual:", n)
	}
}

type wrapper struct {
	n string
	r utils.ReadWrapper
	w utils.WriteWrapper
}

var wrappers = []wrapper{
	{"Zlib", utils.ZlibReader, utils.ZlibWriter},
	{"Nop", utils.NopReadWrapper, utils.NopWriteWrapper},
}

func TestWrappers(t *testing.T) {
	for _, wrapper := range wrappers {
		var buff bytes.Buffer
		var err error
		w := wrapper.w(&buff)
		n, err := w.Write([]byte("test"))
		if err != nil {
			t.Error(wrapper.n, err)
		}
		if n != 4 {
			t.Error(wrapper.n, "expected: 4, actual:", n)
		}
		err = w.Close()
		if err != nil {
			t.Error(wrapper.n, err)
		}
		r, err := wrapper.r(&buff)
		if err != nil {
			t.Error(wrapper.n, err)
		}
		b := make([]byte, 4)
		n, err = r.Read(b)
		if n != 4 {
			t.Error(wrapper.n, "expected: 4, actual:", n)
		}
		if err != nil && err != io.EOF {
			t.Error(wrapper.n, err)
		}
		if !bytes.Equal(b, []byte("test")) {
			t.Errorf("%s, expected: %q, actual: %q", "test", wrapper.n, b)
		}
		n, err = r.Read(b)
		if err != io.EOF {
			t.Error(wrapper.n, err)
		}
	}
}
