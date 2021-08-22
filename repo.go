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
│   ├── files
│   └── recipe
└── 00001/
    ├── chunks/
    │   ├── 000000000000000
    │   ├── 000000000000001
    ├── files
    └── recipe
```
*/

package main

import (
	"encoding/gob"
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

var chunkSize = 8 << 10

type File struct {
	Path string
	Size int64
}

func Commit(source string, repo string) {
	latest := GetLastVersion(repo)
	new := latest + 1
	newPath := path.Join(repo, fmt.Sprintf("%05d", new))
	newChunkPath := path.Join(newPath, "chunks")
	newFilesPath := path.Join(newPath, "files")
	os.Mkdir(newPath, 0775)
	os.Mkdir(newChunkPath, 0775)
	newChunks := make(chan []byte, 16)
	oldChunks := make(chan []byte, 16)
	files := ListFiles(source)
	go LoadChunks(repo, oldChunks)
	go ReadFiles(files, newChunks)
	StoreChunks(newChunkPath, newChunks)
	StoreFiles(newFilesPath, files)
	fmt.Println(files)
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

func ListFiles(path string) []File {
	var files []File
	err := filepath.Walk(path,
		func(p string, i fs.FileInfo, err error) error {
			if err != nil {
				log.Println(err)
				return err
			}
			if i.IsDir() {
				return nil
			}
			files = append(files, File{p, i.Size()})
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	return files
}

func ReadFiles(files []File, chunks chan<- []byte) {
	var buff []byte
	var prev, read = chunkSize, 0

	for _, f := range files {
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

func StoreFiles(dest string, files []File) {
	err := writeFile(dest, files)
	if err != nil {
		log.Println(err)
	}
}

func LoadFiles(repo string) []File {
	files := make([]File, 0)
	err := readFile(repo, &files)
	if err != nil {
		log.Println(err)
	}
	return files
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

func writeFile(filePath string, object interface{}) error {
	file, err := os.Create(filePath)
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(object)
	}
	file.Close()
	return err
}

func readFile(filePath string, object interface{}) error {
	file, err := os.Open(filePath)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(object)
	}
	file.Close()
	return err
}
