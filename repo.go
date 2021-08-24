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

type Repo struct {
	path string
}

type File struct {
	Path string
	Size int64
}

func NewRepo(path string) *Repo {
	os.MkdirAll(path, 0775)
	return &Repo{path}
}

func (r *Repo) Commit(source string) {
	versions := r.loadVersions()
	newVersion := len(versions)
	newPath := path.Join(r.path, fmt.Sprintf(versionFmt, newVersion))
	newChunkPath := path.Join(newPath, chunksName)
	newFilesPath := path.Join(newPath, filesName)
	os.Mkdir(newPath, 0775)
	os.Mkdir(newChunkPath, 0775)
	newChunks := make(chan []byte, 16)
	oldChunks := make(chan Chunk, 16)
	files := listFiles(source)
	go loadChunks(versions, oldChunks)
	go readFiles(files, newChunks)
	// hashes := HashChunks(oldChunks)
	// MatchChunks(newChunks, hashes)
	storeChunks(newChunkPath, newChunks)
	storeFiles(newFilesPath, files)
	fmt.Println(files)
}

func (r *Repo) loadVersions() []string {
	versions := make([]string, 0)
	files, err := os.ReadDir(r.path)
	if err != nil {
		log.Fatalln(err)
	}
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		versions = append(versions, path.Join(r.path, f.Name()))
	}
	return versions
}

func listFiles(path string) []File {
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

func readFiles(files []File, chunks chan<- []byte) {
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

func storeFiles(dest string, files []File) {
	err := writeFile(dest, files)
	if err != nil {
		log.Println(err)
	}
}

func loadFiles(path string) []File {
	files := make([]File, 0)
	err := readFile(path, &files)
	if err != nil {
		log.Println(err)
	}
	return files
}

func printChunks(chunks <-chan []byte) {
	for c := range chunks {
		fmt.Println(c)
	}
}

func storeChunks(dest string, chunks <-chan []byte) {
	i := 0
	for c := range chunks {
		path := path.Join(dest, fmt.Sprintf(chunkIdFmt, i))
		err := os.WriteFile(path, c, 0664)
		if err != nil {
			log.Println(err)
		}
		i++
	}
}

func loadChunks(versions []string, chunks chan<- Chunk) {
	for i, v := range versions {
		p := path.Join(v, chunksName)
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
				Id: &ChunkId{
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

func hashChunks(chunks <-chan Chunk) map[uint64]ChunkId {
	hashes := make(map[uint64]ChunkId)
	hasher := hash.Hash64(rabinkarp64.New())
	for c := range chunks {
		hasher.Reset()
		hasher.Write(c.Value)
		h := hasher.Sum64()
		hashes[h] = *c.Id
	}
	return hashes
}

func (r *Repo) matchChunks(chunks <-chan []byte, hashes map[uint64]ChunkId) []Chunk {
	hasher := rabinkarp64.New()
	hasher.Write(<-chunks)
	recipe := make([]Chunk, 0)

	var i uint64
	var offset, prefill, postfill int
	var exists bool
	var chunkId ChunkId
	for c := range chunks {
		buff := make([]byte, 0)
		// Pre fill the window with the rest of the previous chunk
		for prefill = 0; prefill < offset; prefill++ {
			hasher.Roll(c[prefill])
		}
		// Fill the window with the current chunk and match hash byte by byte
		for ; offset < len(c); offset++ {
			h := hasher.Sum64()
			chunkId, exists = hashes[h]
			if exists {
				// log.Printf("Found existing chunk: New{id:%d, offset:%d} Old%d\n", i, offset, chunkId)
				break
			}
			hasher.Roll(c[offset])
			buff = append(buff, c[offset])
		}
		// Fill the window with the rest of the current chunk if it matched early
		for postfill = offset; postfill < len(c); postfill++ {
			hasher.Roll(c[postfill])
		}
		if len(buff) > 0 {
			recipe = append(recipe, Chunk{Value: buff})
		}
		if exists {
			recipe = append(recipe, Chunk{Id: &chunkId})
		}
		offset %= chunkSize
		i++
	}
	return recipe
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
