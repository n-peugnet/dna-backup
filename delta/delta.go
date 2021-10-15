/* Copyright (C) 2021 Nicolas Peugnet <n.peugnet@free.fr>

   This file is part of dna-backup.

   dna-backup is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   dna-backup is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with dna-backup.  If not, see <https://www.gnu.org/licenses/>. */

package delta

import (
	"fmt"
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
		return fmt.Errorf("source read all: %s", err)
	}
	patchBuf, err := io.ReadAll(patch)
	if err != nil {
		return fmt.Errorf("patch read all: %s", err)
	}
	targetBuf, err := fdelta.Apply(sourceBuf, patchBuf)
	if err != nil {
		return fmt.Errorf("apply patch: %s", err)
	}
	_, err = target.Write(targetBuf)
	return err
}
