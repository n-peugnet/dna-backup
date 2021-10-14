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
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"

	"github.com/n-peugnet/dna-backup/dna"
	"github.com/n-peugnet/dna-backup/logger"
	"github.com/n-peugnet/dna-backup/utils"
)

type Version struct {
	Chunks uint64
	Recipe uint64
	Files  uint64
}

func (r *Repo) ExportDir(dest string, trackSize int) {
	r.Init()
	versions := make([]Version, len(r.versions))
	chunks := r.loadChunks(r.versions)
	for i := range versions {
		var count int64
		var content bytes.Buffer // replace with a reader capable of switching files
		var recipe, fileList []byte
		var err error
		tracker := dna.NewWriter(&content, trackSize)
		counter := utils.NewWriteCounter(tracker)
		compressor := zlib.NewWriter(counter)
		for _, c := range chunks[i] {
			n, err := io.Copy(compressor, c.Reader())
			if err != nil {
				logger.Error(err)
			}
			count += n
		}
		compressor.Close()
		tracker.Close()
		readDelta(r.versions[i], recipeName, utils.NopReadWrapper, func(rc io.ReadCloser) {
			recipe, err = io.ReadAll(rc)
			if err != nil {
				logger.Error("load recipe ", err)
			}
		})
		readDelta(r.versions[i], filesName, utils.NopReadWrapper, func(rc io.ReadCloser) {
			fileList, err = io.ReadAll(rc)
			if err != nil {
				logger.Error("load files ", err)
			}
		})
		versions[i] = Version{
			uint64(counter.Count()),
			uint64(len(recipe)),
			uint64(len(fileList)),
		}
		header := versions[i].createHeader()
		logger.Info(header)
	}
}

func (v Version) createHeader() []byte {
	buf := make([]byte, binary.MaxVarintLen64*3)
	i := 0
	for _, x := range []uint64{v.Chunks, v.Recipe, v.Files} {
		n := binary.PutUvarint(buf[i:], x)
		i += n
	}
	return buf[:i]
}
