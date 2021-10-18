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

package repo

import (
	"io"

	"github.com/n-peugnet/dna-backup/dna"
	"github.com/n-peugnet/dna-backup/logger"
	"github.com/n-peugnet/dna-backup/utils"
)

func (r *Repo) ExportDir(dest string, trackSize int) {
	r.Init()
	exporter := dna.New(dest, 96, trackSize, 10000, utils.ZlibWriter, utils.ZlibReader)
	chunks := r.loadChunks(r.versions)
	for i := range r.versions {
		var err error
		input, end := exporter.VersionInput()
		if len(chunks[i]) > 0 {
			for _, c := range chunks[i] {
				_, err := io.Copy(input.Chunks, c.Reader())
				if err != nil {
					logger.Error(err)
				}
			}
			input.Chunks.Close()
		}
		readDelta(r.versions[i], recipeName, r.chunkReadWrapper, func(rc io.ReadCloser) {
			_, err = io.Copy(input.Recipe, rc)
			if err != nil {
				logger.Error("load recipe ", err)
			}
			input.Recipe.Close()
		})
		readDelta(r.versions[i], filesName, r.chunkReadWrapper, func(rc io.ReadCloser) {
			_, err = io.Copy(input.Files, rc)
			if err != nil {
				logger.Error("load files ", err)
			}
			input.Files.Close()
		})
		<-end
	}
}
