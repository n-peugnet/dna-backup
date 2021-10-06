package delta

import (
	"io"

	"github.com/gabstv/go-bsdiff/pkg/bsdiff"
	"github.com/gabstv/go-bsdiff/pkg/bspatch"
	"github.com/mdvan/fdelta"
)

type Differ interface {
	Diff(source io.Reader, target io.Reader, patch io.Writer) error
}

type Patcher interface {
	Patch(source io.Reader, target io.Writer, patch io.Reader) error
}

type Bsdiff struct{}

func (Bsdiff) Diff(source io.Reader, target io.Reader, patch io.Writer) error {
	return bsdiff.Reader(source, target, patch)
}

func (Bsdiff) Patch(source io.Reader, target io.Writer, patch io.Reader) error {
	return bspatch.Reader(source, target, patch)
}

type Fdelta struct{}

func (Fdelta) Diff(source io.Reader, target io.Reader, patch io.Writer) error {
	sourceBuf, err := io.ReadAll(source)
	if err != nil {
		return err
	}
	targetBuf, err := io.ReadAll(target)
	if err != nil {
		return err
	}
	_, err = patch.Write(fdelta.Create(sourceBuf, targetBuf))
	return err
}

func (Fdelta) Patch(source io.Reader, target io.Writer, patch io.Reader) error {
	sourceBuf, err := io.ReadAll(source)
	if err != nil {
		return err
	}
	patchBuf, err := io.ReadAll(patch)
	if err != nil {
		return err
	}
	targetBuf, err := fdelta.Apply(sourceBuf, patchBuf)
	if err != nil {
		return err
	}
	_, err = target.Write(targetBuf)
	return err
}
