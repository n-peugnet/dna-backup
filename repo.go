package main

import (
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

func ListFiles(path string, files chan<- File) {
	err := filepath.Walk(path,
		func(p string, i fs.FileInfo, err error) error {
			if err != nil {
				log.Println(err)
				return err
			}
			if i.IsDir() {
				return nil
			}
			var file = File{p, i.Size()}
			files <- file
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	close(files)
}

func ReadFiles(files <-chan File, chunks chan<- []byte) {
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

func PrintChunks(chunks <-chan []byte) {
	for c := range chunks {
		fmt.Println(c)
	}
}

func StoreChunks(dest string, chunks <-chan []byte) {
	i := 0
	for c := range chunks {
		path := path.Join(dest, fmt.Sprintf("%015d", i))
		os.WriteFile(path, c, 0664)
		i++
	}
}

func LoadChunks(repo string, chunks chan<- []byte) {
	err := filepath.WalkDir(repo,
		func(p string, e fs.DirEntry, err error) error {
			if err != nil {
				log.Println(err)
				return err
			}
			if e.IsDir() {
				return nil
			}
			buff, err := os.ReadFile(p)
			chunks <- buff
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	close(chunks)
}
