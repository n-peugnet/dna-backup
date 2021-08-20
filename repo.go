package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
)

const (
	chunkSize = 8 << 10
)

type File struct {
	Path string
	Size int64
}

func ListFiles(path string, files chan File) {
	err := filepath.Walk(path,
		func(path string, i fs.FileInfo, err error) error {
			if err != nil {
				log.Println(err)
				return err
			}
			if i.IsDir() {
				return nil
			}
			var file = File{path, i.Size()}
			files <- file
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	close(files)
}

func ReadFiles(files chan File, chunks chan []byte) {
	var buff []byte
	var prev, read = chunkSize, 0

	for f := range files {
		file, err := os.Open(f.Path)
		if err != nil {
			log.Println(err)
			continue
		}
		for err != io.EOF {
			if prev == chunkSize {
				buff = make([]byte, chunkSize)
				prev, err = file.Read(buff)
			} else {
				read, err = file.Read(buff[prev:])
				prev += read
			}
			if err != nil && err != io.EOF {
				log.Println(err)
			}
			if prev == chunkSize {
				chunks <- buff
			}
		}
	}
	chunks <- buff
	close(chunks)
}

func PrintChunks(chunks chan []byte) {
	for c := range chunks {
		fmt.Println(c)
	}
}

func DumpChunks(dest string, chunks chan []byte) {
	for c := range chunks {
		sum := sha1.Sum(c)
		path := path.Join(dest, hex.EncodeToString(sum[:]))
		os.WriteFile(path, c, 0664)
	}
}
