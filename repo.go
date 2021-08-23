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
	"hash"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/chmduquesne/rollinghash/rabinkarp64"
)

var chunkSize = 8 << 10

type File struct {
	Path string
	Size int64
}

type ChunkId struct {
	Ver int
	Idx uint64
}

type Chunk struct {
	Id    ChunkId
	Value []byte
}

func Commit(source string, repo string) {
	versions := LoadVersions(repo)
	newVersion := len(versions)
	newPath := path.Join(repo, fmt.Sprintf("%05d", newVersion))
	newChunkPath := path.Join(newPath, "chunks")
	// newFilesPath := path.Join(newPath, "files")
	os.Mkdir(newPath, 0775)
	os.Mkdir(newChunkPath, 0775)
	newChunks := make(chan []byte, 16)
	oldChunks := make(chan Chunk, 16)
	files := ListFiles(source)
	go LoadChunks(versions, oldChunks)
	go ReadFiles(files, newChunks)
	hashes := HashChunks(oldChunks)
	MatchChunks(newChunks, hashes)
	// StoreChunks(newChunkPath, newChunks)
	// StoreFiles(newFilesPath, files)
	fmt.Println(files)
}

func LoadVersions(repo string) []string {
	versions := make([]string, 0)
	files, err := os.ReadDir(repo)
	if err != nil {
		log.Fatalln(err)
	}
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		versions = append(versions, path.Join(repo, f.Name()))
	}
	return versions
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

func LoadFiles(path string) []File {
	files := make([]File, 0)
	err := readFile(path, &files)
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
		err := os.WriteFile(path, c, 0664)
		if err != nil {
			log.Println(err)
		}
		i++
	}
}

func LoadChunks(versions []string, chunks chan<- Chunk) {
	for i, v := range versions {
		p := path.Join(v, "chunks")
		entries, err := os.ReadDir(p)
		if err != nil {
			log.Printf("Error reading version '%05d' in '%s' chunks: %s", i, v, err)
		}
		for j, e := range entries {
			if e.IsDir() {
				continue
			}
			f := path.Join(p, e.Name())
			buff, err := os.ReadFile(f)
			if err != nil {
				log.Printf("Error reading chunk '%s': %s", f, err.Error())
			}
			c := Chunk{
				Id: ChunkId{
					Ver: i,
					Idx: uint64(j),
				},
				Value: buff,
			}
			chunks <- c
		}
	}
	close(chunks)
}

func HashChunks(chunks <-chan Chunk) map[uint64]ChunkId {
	hashes := make(map[uint64]ChunkId)
	hasher := hash.Hash64(rabinkarp64.New())
	for c := range chunks {
		hasher.Reset()
		hasher.Write(c.Value)
		h := hasher.Sum64()
		hashes[h] = c.Id
	}
	return hashes
}

func MatchChunks(chunks <-chan []byte, hashes map[uint64]ChunkId) {
	hasher := rabinkarp64.New()
	hasher.Write(<-chunks)

	var i uint64
	var offset int
	var prefill int
	var postfill int
	for c := range chunks {
		// Pre fill the window with the rest of the previous chunk
		for prefill = 0; prefill < offset; prefill++ {
			hasher.Roll(c[prefill])
		}
		// Fill the window with the current chunk and match hash byte by byte
		for ; offset < len(c); offset++ {
			h := hasher.Sum64()
			chunk, exists := hashes[h]
			if exists {
				fmt.Printf("Found existing chunk: New{id:%d, offset:%d} Old%d\n", i, offset, chunk)
				break
			}
			hasher.Roll(c[offset])
		}
		// Fill the window with the rest of the current chunk if it matched early
		for postfill = offset; postfill < len(c); postfill++ {
			hasher.Roll(c[postfill])
		}
		offset %= chunkSize
		i++
	}
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
