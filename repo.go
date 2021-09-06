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
	"bytes"
	"encoding/gob"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"

	"github.com/chmduquesne/rollinghash/rabinkarp64"
)

type FingerprintMap map[uint64]*ChunkId
type SketchMap map[uint64][]*ChunkId

type Repo struct {
	path          string
	chunkSize     int
	sketchWSize   int
	sketchSfCount int
	sketchFCount  int
	pol           rabinkarp64.Pol
	differ        Differ
	patcher       Patcher
	fingerprints  FingerprintMap
	sketches      SketchMap
}

type File struct {
	Path string
	Size int64
}

func NewRepo(path string) *Repo {
	err := os.MkdirAll(path, 0775)
	// if err != nil {
	// 	log.Panicln(err)
	// }
	var seed int64 = 1
	p, err := rabinkarp64.RandomPolynomial(seed)
	if err != nil {
		log.Panicln(err)
	}
	return &Repo{
		path:          path,
		chunkSize:     8 << 10,
		sketchWSize:   32,
		sketchSfCount: 3,
		sketchFCount:  4,
		pol:           p,
		differ:        &Bsdiff{},
		patcher:       &Bsdiff{},
		fingerprints:  make(FingerprintMap),
		sketches:      make(SketchMap),
	}
}

func (r *Repo) Differ() Differ {
	return r.differ
}

func (r *Repo) Patcher() Patcher {
	return r.patcher
}

func (r *Repo) Commit(source string) {
	versions := r.loadVersions()
	newVersion := len(versions)
	newPath := path.Join(r.path, fmt.Sprintf(versionFmt, newVersion))
	newChunkPath := path.Join(newPath, chunksName)
	newFilesPath := path.Join(newPath, filesName)
	newRecipePath := path.Join(newPath, recipeName)
	os.Mkdir(newPath, 0775)
	os.Mkdir(newChunkPath, 0775)
	reader, writer := io.Pipe()
	oldChunks := make(chan IdentifiedChunk, 16)
	files := listFiles(source)
	go r.loadChunks(versions, oldChunks)
	go concatFiles(files, writer)
	r.hashChunks(oldChunks)
	recipe := r.matchStream(reader, newVersion)
	storeRecipe(newRecipePath, recipe)
	storeFileList(newFilesPath, files)
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

func (r *Repo) chunkMinLen() int {
	return SuperFeatureSize(r.chunkSize, r.sketchSfCount, r.sketchFCount)
}

func (r *Repo) chunkStream(stream io.Reader, chunks chan<- []byte) {
	var buff []byte
	var prev, read = r.chunkSize, 0
	var err error

	for err != io.EOF {
		if prev == r.chunkSize {
			buff = make([]byte, r.chunkSize)
			prev, err = stream.Read(buff)
		} else {
			read, err = stream.Read(buff[prev:])
			prev += read
		}
		if err != nil && err != io.EOF {
			log.Println(err)
		}
		if prev == r.chunkSize {
			chunks <- buff
		}
	}
	if prev != r.chunkSize {
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

// TODO: use atoi for chunkid
func (r *Repo) loadChunks(versions []string, chunks chan<- IdentifiedChunk) {
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
// stored in an hashmap.
func (r *Repo) hashChunks(chunks <-chan IdentifiedChunk) {
	hasher := rabinkarp64.NewFromPol(r.pol)
	for c := range chunks {
		r.hashAndStoreChunk(c, hasher)
	}
}

func (r *Repo) hashAndStoreChunk(chunk IdentifiedChunk, hasher hash.Hash64) {
	hasher.Reset()
	io.Copy(hasher, chunk.Reader())
	fingerprint := hasher.Sum64()
	sketch, _ := SketchChunk(chunk, r.pol, r.chunkSize, r.sketchWSize, r.sketchSfCount, r.sketchFCount)
	r.storeChunkId(chunk.Id(), fingerprint, sketch)
}

func (r *Repo) storeChunkId(id *ChunkId, fingerprint uint64, sketch []uint64) {
	r.fingerprints[fingerprint] = id
	for _, s := range sketch {
		prev := r.sketches[s]
		if contains(prev, id) {
			continue
		}
		r.sketches[s] = append(prev, id)
	}
}

func contains(s []*ChunkId, id *ChunkId) bool {
	for _, v := range s {
		if v == id {
			return true
		}
	}
	return false
}

func (r *Repo) findSimilarChunk(chunk Chunk) (*ChunkId, bool) {
	var similarChunks = make(map[ChunkId]int)
	var max int
	var similarChunk *ChunkId
	sketch, _ := SketchChunk(chunk, r.pol, r.chunkSize, r.sketchWSize, r.sketchSfCount, r.sketchFCount)
	for _, s := range sketch {
		chunkIds, exists := r.sketches[s]
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

func (r *Repo) tryDeltaEncodeChunk(temp BufferedChunk) (Chunk, bool) {
	id, found := r.findSimilarChunk(temp)
	if found {
		var buff bytes.Buffer
		if err := r.differ.Diff(id.Reader(r), temp.Reader(), &buff); err != nil {
			log.Println("Error trying delta encode chunk:", temp, "with source:", id, ":", err)
		} else {
			return &DeltaChunk{
				repo:   r,
				source: id,
				patch:  buff.Bytes(),
				size:   temp.Len(),
			}, true
		}
	}
	return temp, false
}

// encodeTempChunk first tries to delta-encode the given chunk before attributing
// it an Id and saving it into the fingerprints and sketches maps.
func (r *Repo) encodeTempChunk(temp BufferedChunk, version int, last *uint64) (chunk Chunk, isDelta bool) {
	chunk, isDelta = r.tryDeltaEncodeChunk(temp)
	if isDelta {
		log.Println("Add new delta chunk")
		return
	}
	if chunk.Len() == r.chunkSize {
		id := &ChunkId{Ver: version, Idx: *last}
		*last++
		ic := NewLoadedChunk(id, temp.Bytes())
		hasher := rabinkarp64.NewFromPol(r.pol)
		r.hashAndStoreChunk(ic, hasher)
		ic.Store(r.path)
		log.Println("Add new chunk", id)
		return ic, false
	}
	log.Println("Add new partial chunk of size:", chunk.Len())
	return
}

// encodeTempChunks encodes the current temporary chunks based on the value of the previous one.
// Temporary chunks can be partial. If the current chunk is smaller than the size of a
// super-feature and there exists a previous chunk, then both are merged before attempting
// to delta-encode them.
func (r *Repo) encodeTempChunks(prev BufferedChunk, curr BufferedChunk, version int, last *uint64) []Chunk {
	if reflect.ValueOf(prev).IsNil() {
		c, _ := r.encodeTempChunk(curr, version, last)
		return []Chunk{c}
	} else if curr.Len() < r.chunkMinLen() {
		c, success := r.encodeTempChunk(NewTempChunk(append(prev.Bytes(), curr.Bytes()...)), version, last)
		if success {
			return []Chunk{c}
		} else {
			return []Chunk{prev, curr}
		}
	} else {
		prevD, _ := r.encodeTempChunk(prev, version, last)
		currD, _ := r.encodeTempChunk(curr, version, last)
		return []Chunk{prevD, currD}
	}
}

func (r *Repo) matchStream(stream io.Reader, version int) []Chunk {
	var b byte
	var chunks []Chunk
	var prev *TempChunk
	var last uint64
	bufStream := bufio.NewReaderSize(stream, r.chunkSize*2)
	buff := make([]byte, 0, r.chunkSize*2)
	n, err := io.ReadFull(stream, buff[:r.chunkSize])
	if n < r.chunkSize {
		chunks = append(chunks, NewTempChunk(buff[:n]))
		return chunks
	}
	hasher := rabinkarp64.NewFromPol(r.pol)
	hasher.Write(buff[:n])
	for err != io.EOF {
		h := hasher.Sum64()
		chunkId, exists := r.fingerprints[h]
		if exists {
			if len(buff) > r.chunkSize && len(buff) < r.chunkSize*2 {
				size := len(buff) - r.chunkSize
				temp := NewTempChunk(buff[:size])
				chunks = append(chunks, r.encodeTempChunks(prev, temp, version, &last)...)
				prev = nil
			} else if prev != nil {
				c, _ := r.encodeTempChunk(prev, version, &last)
				chunks = append(chunks, c)
				prev = nil
			}
			log.Printf("Add existing chunk: %d\n", chunkId)
			chunks = append(chunks, NewStoredFile(r, chunkId))
			buff = make([]byte, 0, r.chunkSize*2)
			for i := 0; i < r.chunkSize && err == nil; i++ {
				b, err = bufStream.ReadByte()
				hasher.Roll(b)
				buff = append(buff, b)
			}
			continue
		}
		if len(buff) == r.chunkSize*2 {
			if prev != nil {
				chunk, _ := r.encodeTempChunk(prev, version, &last)
				chunks = append(chunks, chunk)
			}
			prev = NewTempChunk(buff[:r.chunkSize])
			tmp := buff[r.chunkSize:]
			buff = make([]byte, 0, r.chunkSize*2)
			buff = append(buff, tmp...)
		}
		b, err = bufStream.ReadByte()
		if err != io.EOF {
			hasher.Roll(b)
			buff = append(buff, b)
		}
	}
	if len(buff) > 0 {
		var temp *TempChunk
		if len(buff) > r.chunkSize {
			prev = NewTempChunk(buff[:r.chunkSize])
			temp = NewTempChunk(buff[r.chunkSize:])
		} else {
			temp = NewTempChunk(buff)
		}
		chunks = append(chunks, r.encodeTempChunks(prev, temp, version, &last)...)
	}
	return chunks
}

func storeRecipe(dest string, recipe []Chunk) {
	err := writeFile(dest, recipe)
	if err != nil {
		log.Println(err)
	}
}

// mergeTempChunks joins temporary partial chunks from an array of chunks if possible.
// If a chunk is smaller than the size required to calculate a super-feature,
// it is then appended to the previous consecutive temporary chunk if it exists.
func (r *Repo) mergeTempChunks(chunks []Chunk) (ret []Chunk) {
	var prev *TempChunk
	var curr *TempChunk
	for _, c := range chunks {
		tmp, isTmp := c.(*TempChunk)
		if !isTmp {
			if prev != nil && curr.Len() <= SuperFeatureSize(r.chunkSize, r.sketchSfCount, r.sketchFCount) {
				prev.AppendFrom(curr.Reader())
			} else if curr != nil {
				ret = append(ret, curr)
			}
			ret = append(ret, c)
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

func extractTempChunks(chunks []Chunk) (ret []*TempChunk) {
	for _, c := range chunks {
		tmp, isTmp := c.(*TempChunk)
		if isTmp {
			ret = append(ret, tmp)
		}
	}
	return
}

func extractDeltaChunks(chunks []Chunk) (ret []*DeltaChunk) {
	for _, c := range chunks {
		tmp, isDelta := c.(*DeltaChunk)
		if isDelta {
			ret = append(ret, tmp)
		}
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
