package main

import (
	"io"

	"github.com/gabstv/go-bsdiff/pkg/bsdiff"
	"github.com/gabstv/go-bsdiff/pkg/bspatch"
)

type DeltaCodec interface {
	Differ
	Patcher
}

type Differ interface {
	Diff(source io.Reader, target io.Reader, patch io.Writer) error
}

type Patcher interface {
	Patch(source io.Reader, target io.Writer, patch io.Reader) error
}

// TODO: maybe move this in it own file ?
type Bsdiff struct{}

func (*Bsdiff) Diff(source io.Reader, target io.Reader, patch io.Writer) error {
	return bsdiff.Reader(source, target, patch)
}

func (*Bsdiff) Patch(source io.Reader, target io.Writer, patch io.Reader) error {
	return bspatch.Reader(source, target, patch)
}
