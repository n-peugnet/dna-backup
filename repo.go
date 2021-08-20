/*
Manage a deduplicated versionned backups repository.

Sample repository:

```
repo/
├── 00000/
│   ├── chunks/
│   │   ├── 000000000000000
│   │   ├── 000000000000001
│   │   ├── 000000000000002
│   │   ├── 000000000000003
│   ├── dentries
│   └── recipe
└── 00001/
    ├── chunks/
    │   ├── 000000000000000
    │   ├── 000000000000001
    ├── dentries
    └── recipe
```
*/

package main

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

const (
	chunkSize = 8 << 10
)

type File struct {
	Path string
	Size int64
}

func Commit(source string, repo string) {
	latest := GetLastVersion(repo)
	new := latest + 1
	newPath := path.Join(repo, fmt.Sprintf("%05d", new))
	newChunkPath := path.Join(newPath, "chunks")
	os.Mkdir(newPath, 0775)
	os.Mkdir(newChunkPath, 0775)
	files := make(chan File)
	newChunks := make(chan []byte)
	oldChunks := make(chan []byte)
	go LoadChunks(repo, oldChunks)
	go ListFiles(source, files)
	go ReadFiles(files, newChunks)
	StoreChunks(newChunkPath, newChunks)
}

func GetLastVersion(repo string) int {
	v := -1
	files, err := ioutil.ReadDir(repo)
	if err != nil {
		log.Fatalln(err)
	}
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		num, err := strconv.Atoi(f.Name())
		if err != nil {
			log.Println(err)
			continue
		}
		if num > v {
			v = num
		}
	}
	return v
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
