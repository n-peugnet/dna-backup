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

package dna

import (
	"io"

	"github.com/n-peugnet/dna-backup/utils"
)

type writer struct {
	*utils.WriteCounter
	trackSize int
}

func NewWriter(w io.Writer, trackSize int) io.WriteCloser {
	return &writer{
		WriteCounter: utils.NewWriteCounter(w),
		trackSize:    trackSize,
	}
}

func (d *writer) Close() (err error) {
	// add padding for the last track
	padding := make([]byte, d.trackSize-d.Count()%d.trackSize)
	if _, err = d.Write(padding); err != nil {
		return err
	}
	return nil
}
