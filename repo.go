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
	"bufio"
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

type FingerprintMap map[uint64]*ChunkId
type SketchMap map[uint64][]*ChunkId

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
	// newFilesPath := path.Join(newPath, filesName)
	os.Mkdir(newPath, 0775)
	os.Mkdir(newChunkPath, 0775)
	reader, writer := io.Pipe()
	oldChunks := make(chan StoredChunk, 16)
	files := listFiles(source)
	go r.loadChunks(versions, oldChunks)
	go concatFiles(files, writer)
	fingerprints, _ := hashChunks(oldChunks)
	chunks := r.matchStream(reader, fingerprints)
	extractTempChunks(chunks)
	// storeChunks(newChunkPath, newChunks)
	// storeFiles(newFilesPath, files)
	fmt.Println(files)
}

func (r *Repo) loadVersions() []string {
	var versions []string
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

func concatFiles(files []File, stream io.WriteCloser) {
	for _, f := range files {
		file, err := os.Open(f.Path)
		if err != nil {
			log.Printf("Error reading file '%s': %s\n", f.Path, err)
			continue
		}
		io.Copy(stream, file)
	}
	stream.Close()
}

func chunkStream(stream io.Reader, chunks chan<- []byte) {
	var buff []byte
	var prev, read = chunkSize, 0
	var err error

	for err != io.EOF {
		if prev == chunkSize {
			buff = make([]byte, chunkSize)
			prev, err = stream.Read(buff)
		} else {
			read, err = stream.Read(buff[prev:])
			prev += read
		}
		if err != nil && err != io.EOF {
			log.Println(err)
		}
		if prev == chunkSize {
			chunks <- buff
		}
	}
	if prev != chunkSize {
		chunks <- buff[:prev]
	}
	close(chunks)
}

func storeFileList(dest string, files []File) {
	err := writeFile(dest, files)
	if err != nil {
		log.Println(err)
	}
}

func loadFileList(path string) []File {
	var files []File
	err := readFile(path, &files)
	if err != nil {
		log.Println(err)
	}
	return files
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

func (r *Repo) loadChunks(versions []string, chunks chan<- StoredChunk) {
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
			c := NewLoadedChunk(
				&ChunkId{
					Ver: i,
					Idx: uint64(j),
				},
				buff,
			)
			chunks <- c
		}
	}
	close(chunks)
}

// hashChunks calculates the hashes for a channel of chunks.
//
// For each chunk, both a fingerprint (hash over the full content) and a sketch
// (resemblance hash based on maximal values of regions) are calculated and
// stored in an hashmap which are then returned.
func hashChunks(chunks <-chan StoredChunk) (FingerprintMap, SketchMap) {
	fingerprints := make(FingerprintMap)
	sketches := make(SketchMap)
	hasher := hash.Hash64(rabinkarp64.New())
	for c := range chunks {
		hasher.Reset()
		io.Copy(hasher, c.Reader())
		h := hasher.Sum64()
		fingerprints[h] = c.Id()
		sketch, _ := SketchChunk(c, sketchWSize, sketchSfCount, sketchFCount)
		for _, s := range sketch {
			prev := sketches[s]
			if contains(prev, c.Id()) {
				continue
			}
			sketches[s] = append(prev, c.Id())
		}
	}
	return fingerprints, sketches
}

func contains(s []*ChunkId, id *ChunkId) bool {
	for _, v := range s {
		if v == id {
			return true
		}
	}
	return false
}

func findSimilarChunk(chunk Chunk, sketches SketchMap) (*ChunkId, bool) {
	var similarChunks = make(map[ChunkId]int)
	var max int
	var similarChunk *ChunkId
	sketch, _ := SketchChunk(chunk, sketchWSize, sketchSfCount, sketchFCount)
	for _, s := range sketch {
		chunkIds, exists := sketches[s]
		if !exists {
			continue
		}
		for _, id := range chunkIds {
			count := similarChunks[*id]
			count += 1
			log.Printf("Found %d %d time(s)", id, count)
			if count > max {
				similarChunk = id
			}
			similarChunks[*id] = count
		}
	}
	return similarChunk, similarChunk != nil
}

func (r *Repo) matchStream(stream io.Reader, fingerprints FingerprintMap) []Chunk {
	var b byte
	var chunks []Chunk
	bufStream := bufio.NewReaderSize(stream, chunkSize)
	buff := make([]byte, 0, chunkSize*2)
	n, err := io.ReadFull(stream, buff[:chunkSize])
	if n < chunkSize {
		chunks = append(chunks, NewTempChunk(buff[:n]))
		return chunks
	}
	hasher := rabinkarp64.New()
	hasher.Write(buff[:n])
	for err != io.EOF {
		h := hasher.Sum64()
		chunkId, exists := fingerprints[h]
		if exists {
			if len(buff) > chunkSize && len(buff) < chunkSize*2 {
				size := len(buff) - chunkSize
				log.Println("Add new partial chunk of size:", size)
				chunks = append(chunks, NewTempChunk(buff[:size]))
			}
			log.Printf("Add existing chunk: %d\n", chunkId)
			chunks = append(chunks, NewChunkFile(r, chunkId))
			buff = make([]byte, 0, chunkSize*2)
			for i := 0; i < chunkSize && err == nil; i++ {
				b, err = bufStream.ReadByte()
				hasher.Roll(b)
				buff = append(buff, b)
			}
			continue
		}
		if len(buff) == chunkSize*2 {
			log.Println("Add new chunk")
			chunks = append(chunks, NewTempChunk(buff[:chunkSize]))
			tmp := buff[chunkSize:]
			buff = make([]byte, 0, chunkSize*2)
			buff = append(buff, tmp...)
		}
		b, err = bufStream.ReadByte()
		if err != io.EOF {
			hasher.Roll(b)
			buff = append(buff, b)
		}
	}
	if len(buff) > chunkSize {
		log.Println("Add new chunk")
		chunks = append(chunks, NewTempChunk(buff[:chunkSize]))
		log.Println("Add new partial chunk of size:", len(buff)-chunkSize)
		chunks = append(chunks, NewTempChunk(buff[chunkSize:]))
	} else if len(buff) > 0 {
		log.Println("Add new partial chunk of size:", len(buff))
		chunks = append(chunks, NewTempChunk(buff))
	}
	return chunks
}

// extractTempChunks extracts temporary chunks from an array of chunks.
// If a chunk is smaller than the size required to calculate a super-feature,
// it is then appended to the previous consecutive temporary chunk if it exists.
func extractTempChunks(chunks []Chunk) (ret []Chunk) {
	var prev *TempChunk
	var curr *TempChunk
	for _, c := range chunks {
		tmp, isTmp := c.(*TempChunk)
		if !isTmp {
			if prev != nil && curr.Len() <= SuperFeatureSize(chunkSize, sketchSfCount, sketchFCount) {
				prev.AppendFrom(curr.Reader())
			} else if curr != nil {
				ret = append(ret, curr)
			}
			curr = nil
			prev = nil
		} else {
			prev = curr
			curr = tmp
			if prev != nil {
				ret = append(ret, prev)
			}
		}
	}
	if curr != nil {
		ret = append(ret, curr)
	}
	return
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
