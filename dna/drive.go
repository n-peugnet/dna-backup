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
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/n-peugnet/dna-backup/logger"
	"github.com/n-peugnet/dna-backup/utils"
)

type Direction int

const (
	Forward Direction = iota
	Backward
)

type DnaDrive struct {
	poolCount     int
	trackSize     int
	tracksPerPool int
	pools         []Pool
	writeWrapper  utils.WriteWrapper
	readWrapper   utils.ReadWrapper
}

type Pool struct {
	Data       io.ReadWriteCloser
	TrackCount int
}

type Header struct {
	Chunks uint64
	Recipe uint64
	Files  uint64
}

type dnaVersion struct {
	Input  dnaInput
	Output dnaOutput
}

type dnaInput struct {
	Chunks io.WriteCloser
	Recipe io.WriteCloser
	Files  io.WriteCloser
}

type dnaOutput struct {
	Chunks io.ReadCloser
	Recipe io.ReadCloser
	Files  io.ReadCloser
}

func New(
	destination string,
	poolCount int,
	trackSize int,
	tracksPerPool int,
	writeWrapper utils.WriteWrapper,
	readWrapper utils.ReadWrapper,
) *DnaDrive {
	pools := make([]Pool, poolCount)
	os.MkdirAll(destination, 0755)
	for i := range pools {
		path := filepath.Join(destination, fmt.Sprintf("%02d", i))
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			logger.Panic(err)
		}
		stat, err := file.Stat()
		if err != nil {
			logger.Panic(err)
		}
		pools[i] = Pool{file, int(stat.Size()) / trackSize}
	}
	return &DnaDrive{
		poolCount:     poolCount,
		trackSize:     trackSize,
		tracksPerPool: tracksPerPool,
		pools:         pools,
		writeWrapper:  writeWrapper,
		readWrapper:   readWrapper,
	}
}

func (d *DnaDrive) VersionInput() (dnaInput, <-chan bool) {
	rChunks, wChunks := io.Pipe()
	rRecipe, wRecipe := io.Pipe()
	rFiles, wFiles := io.Pipe()
	version := dnaVersion{
		Input:  dnaInput{wChunks, wRecipe, wFiles},
		Output: dnaOutput{rChunks, rRecipe, rFiles},
	}
	end := make(chan bool)
	go d.writeVersion(version.Output, end)
	return version.Input, end
}

func (d *DnaDrive) writeVersion(output dnaOutput, end chan<- bool) {
	var err error
	var recipe, files, version bytes.Buffer
	n := d.write(output.Chunks, d.pools[1:], Forward)
	_, err = io.Copy(&recipe, output.Recipe)
	if err != nil {
		logger.Error("dna export recipe ", err)
	}
	_, err = io.Copy(&files, output.Files)
	if err != nil {
		logger.Error("dna export files ", err)
	}
	header := Header{
		uint64(n),
		uint64(recipe.Len()),
		uint64(files.Len()),
	}
	e := gob.NewEncoder(&version)
	err = e.Encode(header)
	if err != nil {
		logger.Error("dna export version header: ", err)
	}
	logger.Debugf("version len %d", version.Len())
	rest := int64(d.trackSize - version.Len())
	logger.Debugf("version rest %d", rest)
	n, err = io.CopyN(&version, &recipe, rest)
	logger.Debugf("recipe copied in version %d", n)
	rest -= n
	logger.Debugf("version rest %d", rest)
	if err == io.EOF && rest > 0 { // recipe is written to version but there is space left
		n, err = io.CopyN(&version, &files, rest)
		logger.Debugf("files copied in version %d", n)
		rest -= n
		logger.Debugf("version rest %d", rest)
		if err == io.EOF && rest > 0 { // files is writtent to version but there is space left
			// version.Write(make([]byte, rest))
		} else if err != nil { // another error than EOF happened
			logger.Error("dna export files: ", err)
		} else { // files has not been fully written so we write what is left to pools
			d.write(&files, d.pools[1:], Backward)
		}
	} else if err != nil { // another error than EOF happened
		logger.Error("dna export recipe: ", err)
	} else { // recipe has not been fully written so we concat with files and write what is left to pools
		io.Copy(&recipe, &files)
		d.write(&recipe, d.pools[1:], Backward)
	}
	d.write(&version, d.pools[:1], Forward)
	end <- true
}

func (d *DnaDrive) write(r io.Reader, pools []Pool, direction Direction) int64 {
	var err error
	var i, n int
	var count int64
	if direction == Backward {
		i = len(pools) - 1
	}
	for err != io.ErrUnexpectedEOF && err != io.EOF {
		if pools[i].TrackCount == d.tracksPerPool {
			if direction == Backward {
				i--
			} else {
				i++
			}
			if i < 0 || i >= len(pools) {
				logger.Panic("dna export: no space left")
			}
			continue
		}
		buf := make([]byte, d.trackSize)
		n, err = io.ReadFull(r, buf)
		if err == io.EOF {
			break
		}
		logger.Debug("written track:", n, err)
		count += int64(n)
		n, errw := pools[i].Data.Write(buf)
		if errw != nil {
			logger.Error("dna export: pool %d: %d/%d bytes written: %s", i, n, len(buf), errw)
		}
		pools[i].TrackCount++
	}
	return count
}
