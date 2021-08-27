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
	oldChunks := make(chan Chunk, 16)
	files := listFiles(source)
	go r.loadChunks(versions, oldChunks)
	go concatFiles(files, writer)
	fingerprints, _ := hashChunks(oldChunks)
	chunks := r.matchStream(reader, fingerprints)
	extractNewChunks(chunks)
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
		chunks <- buff
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

func (r *Repo) loadChunks(versions []string, chunks chan<- Chunk) {
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
				Repo: r,
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

// hashChunks calculates the hashes for a channel of chunks.
//
// For each chunk, both a fingerprint (hash over the full content) and a sketch
// (resemblance hash based on maximal values of regions) are calculated and
// stored in an hashmap which are then returned.
func hashChunks(chunks <-chan Chunk) (FingerprintMap, SketchMap) {
	fingerprints := make(FingerprintMap)
	sketches := make(SketchMap)
	hasher := hash.Hash64(rabinkarp64.New())
	for c := range chunks {
		hasher.Reset()
		hasher.Write(c.Value)
		h := hasher.Sum64()
		fingerprints[h] = c.Id
		sketch, _ := SketchChunk(c, 32, 3, 4)
		for _, s := range sketch {
			sketches[s] = append(sketches[s], c.Id)
		}
	}
	return fingerprints, sketches
}

func findSimilarChunks(chunks []Chunk, sketches SketchMap) {
	for _, c := range chunks {
		sketch, _ := SketchChunk(c, 32, 3, 4)
		for _, s := range sketch {
			chunkId, exists := sketches[s]
			if exists {
				log.Println("Found similar chunks: ", chunkId)
			}
		}
	}
}

func readChunk(stream io.Reader) ([]byte, error) {
	buff := make([]byte, chunkSize)
	_, err := io.ReadFull(stream, buff)
	return buff, err
}

func (r *Repo) matchStream(stream io.Reader, fingerprints FingerprintMap) []Chunk {
	var b byte
	var chunks []Chunk
	bufStream := bufio.NewReaderSize(stream, chunkSize)
	buff, err := readChunk(bufStream)
	if err == io.EOF {
		chunks = append(chunks, Chunk{Repo: r, Value: buff})
		return chunks
	}
	hasher := rabinkarp64.New()
	hasher.Write(buff)
	buff = make([]byte, 0, chunkSize)
	for err != io.EOF {
		h := hasher.Sum64()
		chunkId, exists := fingerprints[h]
		if exists {
			if len(buff) > 0 {
				log.Printf("Add new partial chunk of size: %d\n", len(buff))
				chunks = append(chunks, Chunk{Repo: r, Value: buff[:chunkSize]})
			}
			log.Printf("Add existing chunk: %d\n", chunkId)
			chunks = append(chunks, Chunk{Repo: r, Id: chunkId})
			buff = make([]byte, 0, chunkSize)
			for i := 0; i < chunkSize && err == nil; i++ {
				b, err = bufStream.ReadByte()
				hasher.Roll(b)
			}
			continue
		}
		if len(buff) == chunkSize {
			log.Println("Add new chunk")
			chunks = append(chunks, Chunk{Repo: r, Value: buff})
			buff = make([]byte, 0, chunkSize)
		}
		b, err = bufStream.ReadByte()
		if err != io.EOF {
			hasher.Roll(b)
			buff = append(buff, b)
		}
	}
	if len(buff) > 0 {
		chunks = append(chunks, Chunk{Repo: r, Value: buff})
	}
	return chunks
}

// extractNewChunks extracts new chunks from an array of chunks and
// returns them in an array of consecutive new chunk's array
func extractNewChunks(chunks []Chunk) (ret [][]Chunk) {
	var i int
	ret = append(ret, nil)
	for _, c := range chunks {
		if c.isStored() {
			if len(ret[i]) != 0 {
				i++
				ret = append(ret, nil)
			}
		} else {
			ret[i] = append(ret[i], c)
		}
	}
	if len(ret[i]) == 0 {
		ret = ret[:i]
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
