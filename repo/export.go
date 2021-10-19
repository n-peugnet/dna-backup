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

	"github.com/n-peugnet/dna-backup/export"
	"github.com/n-peugnet/dna-backup/logger"
	"github.com/n-peugnet/dna-backup/utils"
)

func (r *Repo) Export(exporter export.Exporter) {
	r.Init()
	chunks := r.loadChunks(r.versions)
	for i := range r.versions {
		var err error
		end := make(chan bool)
		input := exporter.ExportVersion(end)
		go exportChunks(chunks[i], r.chunkWriteWrapper, input.Chunks)
		readDelta(r.versions[i], recipeName, utils.NopReadWrapper, func(rc io.ReadCloser) {
			_, err = io.Copy(input.Recipe, rc)
			if err != nil {
				logger.Error("load recipe ", err)
			}
			if err = input.Recipe.Close(); err != nil {
				logger.Error("export recipe ", err)
			}
		})
		readDelta(r.versions[i], filesName, utils.NopReadWrapper, func(rc io.ReadCloser) {
			_, err = io.Copy(input.Files, rc)
			if err != nil {
				logger.Error("load files ", err)
			}
			if err = input.Files.Close(); err != nil {
				logger.Error("export files ", err)
			}
		})
		<-end
	}
}

func exportChunks(chunks []IdentifiedChunk, wrapper utils.WriteWrapper, input io.WriteCloser) {
	if len(chunks) > 0 {
		compressed := wrapper(input)
		for _, c := range chunks {
			_, err := io.Copy(compressed, c.Reader())
			if err != nil {
				logger.Error(err)
			}
		}
		compressed.Close()
	}
	input.Close()
}
