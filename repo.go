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
	"path/filepath"
	"reflect"
	"strings"

	"github.com/chmduquesne/rollinghash/rabinkarp64"
	"github.com/n-peugnet/dna-backup/cache"
	"github.com/n-peugnet/dna-backup/sketch"
	"github.com/n-peugnet/dna-backup/utils"
)

type FingerprintMap map[uint64]*ChunkId
type SketchMap map[uint64][]*ChunkId

type Repo struct {
	path              string
	chunkSize         int
	sketchWSize       int
	sketchSfCount     int
	sketchFCount      int
	pol               rabinkarp64.Pol
	differ            Differ
	patcher           Patcher
	fingerprints      FingerprintMap
	sketches          SketchMap
	chunkCache        cache.Cacher
	chunkReadWrapper  func(r io.Reader) (io.ReadCloser, error)
	chunkWriteWrapper func(w io.Writer) io.WriteCloser
}

type File struct {
	Path string
	Size int64
}

func NewRepo(path string) *Repo {
	err := os.MkdirAll(path, 0775)
	if err != nil {
		log.Panicln(err)
	}
	var seed int64 = 1
	p, err := rabinkarp64.RandomPolynomial(seed)
	if err != nil {
		log.Panicln(err)
	}
	return &Repo{
		path:              path,
		chunkSize:         8 << 10,
		sketchWSize:       32,
		sketchSfCount:     3,
		sketchFCount:      4,
		pol:               p,
		differ:            &Bsdiff{},
		patcher:           &Bsdiff{},
		fingerprints:      make(FingerprintMap),
		sketches:          make(SketchMap),
		chunkCache:        cache.NewFifoCache(1000),
		chunkReadWrapper:  utils.ZlibReader,
		chunkWriteWrapper: utils.ZlibWriter,
	}
}

func (r *Repo) Differ() Differ {
	return r.differ
}

func (r *Repo) Patcher() Patcher {
	return r.patcher
}

func (r *Repo) Commit(source string) {
	source = utils.TrimTrailingSeparator(source)
	versions := r.loadVersions()
	newVersion := len(versions) // TODO: add newVersion functino
	newPath := filepath.Join(r.path, fmt.Sprintf(versionFmt, newVersion))
	newChunkPath := filepath.Join(newPath, chunksName)
	newFilesPath := filepath.Join(newPath, filesName)
	newRecipePath := filepath.Join(newPath, recipeName)
	os.Mkdir(newPath, 0775)      // TODO: handle errors
	os.Mkdir(newChunkPath, 0775) // TODO: handle errors
	reader, writer := io.Pipe()
	oldChunks := make(chan IdentifiedChunk, 16)
	files := listFiles(source)
	go r.loadChunks(versions, oldChunks)
	go concatFiles(files, writer)
	r.hashChunks(oldChunks)
	recipe := r.matchStream(reader, newVersion)
	storeRecipe(newRecipePath, recipe)
	storeFileList(newFilesPath, unprefixFiles(files, source))
	fmt.Println(files)
}

func (r *Repo) Restore(destination string) {
	versions := r.loadVersions()
	latest := versions[len(versions)-1]
	latestFilesPath := filepath.Join(latest, filesName)
	latestRecipePath := filepath.Join(latest, recipeName)
	files := loadFileList(latestFilesPath)
	recipe := loadRecipe(latestRecipePath)
	reader, writer := io.Pipe()
	go r.restoreStream(writer, recipe)
	bufReader := bufio.NewReaderSize(reader, r.chunkSize*2)
	for _, file := range files {
		filePath := filepath.Join(destination, file.Path)
		dir := filepath.Dir(filePath)
		os.MkdirAll(dir, 0775)      // TODO: handle errors
		f, _ := os.Create(filePath) // TODO: handle errors
		n, err := io.CopyN(f, bufReader, file.Size)
		if err != nil {
			log.Printf("Error storing file content for '%s', written %d/%d bytes: %s\n", filePath, n, file.Size, err)
		}
		if err := f.Close(); err != nil {
			log.Printf("Error closing restored file '%s': %s\n", filePath, err)
		}
	}
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
		versions = append(versions, filepath.Join(r.path, f.Name()))
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

func unprefixFiles(files []File, prefix string) (ret []File) {
	ret = make([]File, len(files))
	preSize := len(prefix)
	for i, f := range files {
		if !strings.HasPrefix(f.Path, prefix) {
			log.Println("Warning", f.Path, "is not prefixed by", prefix)
		} else {
			f.Path = f.Path[preSize:]
		}
		ret[i] = f
	}
	return
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

func storeFileList(dest string, files []File) {
	file, err := os.Create(dest)
	if err == nil {
		encoder := gob.NewEncoder(file)
		err = encoder.Encode(files)
	}
	if err != nil {
		log.Panicln(err)
	}
	if err = file.Close(); err != nil {
		log.Panicln(err)
	}
}

func loadFileList(path string) []File {
	var files []File
	file, err := os.Open(path)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&files)
	}
	if err != nil {
		log.Panicln(err)
	}
	if err = file.Close(); err != nil {
		log.Panicln(err)
	}
	return files
}

func (r *Repo) StoreChunkContent(id *ChunkId, reader io.Reader) error {
	path := id.Path(r.path)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Error creating chunk for '%s'; %s\n", path, err)
	}
	wrapper := r.chunkWriteWrapper(file)
	n, err := io.Copy(wrapper, reader)
	if err != nil {
		return fmt.Errorf("Error writing chunk content for '%s', written %d bytes: %s\n", path, n, err)
	}
	if err := wrapper.Close(); err != nil {
		return fmt.Errorf("Error closing write wrapper for '%s': %s\n", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("Error closing chunk for '%s': %s\n", path, err)
	}
	return nil
}

// LoadChunkContent loads a chunk from the repo.
// If the chunk is in cache, get it from cache, else read it from drive.
func (r *Repo) LoadChunkContent(id *ChunkId) *bytes.Reader {
	value, exists := r.chunkCache.Get(id)
	if !exists {
		path := id.Path(r.path)
		f, err := os.Open(path)
		if err != nil {
			log.Printf("Cannot open chunk '%s': %s\n", path, err)
		}
		wrapper, err := r.chunkReadWrapper(f)
		if err != nil {
			log.Printf("Cannot create read wrapper for chunk '%s': %s\n", path, err)
		}
		value, err = io.ReadAll(wrapper)
		if err != nil {
			log.Panicf("Could not read from chunk '%s': %s\n", path, err)
		}
		if err = f.Close(); err != nil {
			log.Printf("Could not close chunk '%s': %s\n", path, err)
		}
		r.chunkCache.Set(id, value)
	}
	return bytes.NewReader(value)
}

// TODO: use atoi for chunkid
func (r *Repo) loadChunks(versions []string, chunks chan<- IdentifiedChunk) {
	for i, v := range versions {
		p := filepath.Join(v, chunksName)
		entries, err := os.ReadDir(p)
		if err != nil {
			log.Printf("Error reading version '%05d' in '%s' chunks: %s", i, v, err)
		}
		for j, e := range entries {
			if e.IsDir() {
				continue
			}
			f := filepath.Join(p, e.Name())
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

func (r *Repo) chunkMinLen() int {
	return sketch.SuperFeatureSize(r.chunkSize, r.sketchSfCount, r.sketchFCount)
}

// hashChunks calculates the hashes for a channel of chunks.
//
// For each chunk, both a fingerprint (hash over the full content) and a sketch
// (resemblance hash based on maximal values of regions) are calculated and
// stored in an hashmap.
func (r *Repo) hashChunks(chunks <-chan IdentifiedChunk) {
	hasher := rabinkarp64.NewFromPol(r.pol)
	for c := range chunks {
		r.hashAndStoreChunk(c.GetId(), c.Reader(), hasher)
	}
}

func (r *Repo) hashAndStoreChunk(id *ChunkId, reader io.Reader, hasher hash.Hash64) {
	var chunk bytes.Buffer
	hasher.Reset()
	reader = io.TeeReader(reader, &chunk)
	io.Copy(hasher, reader)
	fingerprint := hasher.Sum64()
	sketch, _ := sketch.SketchChunk(&chunk, r.pol, r.chunkSize, r.sketchWSize, r.sketchSfCount, r.sketchFCount)
	r.storeChunkId(id, fingerprint, sketch)
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
	sketch, _ := sketch.SketchChunk(chunk.Reader(), r.pol, r.chunkSize, r.sketchWSize, r.sketchSfCount, r.sketchFCount)
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
		if err := r.differ.Diff(r.LoadChunkContent(id), temp.Reader(), &buff); err != nil {
			log.Println("Error trying delta encode chunk:", temp, "with source:", id, ":", err)
		} else {
			return &DeltaChunk{
				repo:   r,
				Source: id,
				Patch:  buff.Bytes(),
				Size:   temp.Len(),
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
		hasher := rabinkarp64.NewFromPol(r.pol)
		r.hashAndStoreChunk(id, temp.Reader(), hasher)
		err := r.StoreChunkContent(id, temp.Reader())
		if err != nil {
			log.Println(err)
		}
		log.Println("Add new chunk", id)
		return NewStoredChunk(r, id), false
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
	var err error
	bufStream := bufio.NewReaderSize(stream, r.chunkSize*2)
	buff := make([]byte, r.chunkSize, r.chunkSize*2)
	if n, err := io.ReadFull(stream, buff); n < r.chunkSize {
		if err == io.EOF {
			chunks = append(chunks, NewTempChunk(buff[:n]))
			return chunks
		} else {
			log.Panicf("Error Read only %d bytes with error '%s'\n", n, err)
		}
	}
	hasher := rabinkarp64.NewFromPol(r.pol)
	hasher.Write(buff)
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
			chunks = append(chunks, NewStoredChunk(r, chunkId))
			buff = make([]byte, 0, r.chunkSize*2)
			for i := 0; i < r.chunkSize && err == nil; i++ {
				b, err = bufStream.ReadByte()
				if err != io.EOF {
					hasher.Roll(b)
					buff = append(buff, b)
				}
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
			buff = make([]byte, r.chunkSize, r.chunkSize*2)
			copy(buff, tmp)
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
			if prev != nil {
				chunk, _ := r.encodeTempChunk(prev, version, &last)
				chunks = append(chunks, chunk)
			}
			prev = NewTempChunk(buff[:r.chunkSize])
			temp = NewTempChunk(buff[r.chunkSize:])
		} else {
			temp = NewTempChunk(buff)
		}
		chunks = append(chunks, r.encodeTempChunks(prev, temp, version, &last)...)
	}
	return chunks
}

func (r *Repo) restoreStream(stream io.WriteCloser, recipe []Chunk) {
	for _, c := range recipe {
		if rc, isRepo := c.(RepoChunk); isRepo {
			rc.SetRepo(r)
		}
		if n, err := io.Copy(stream, c.Reader()); err != nil {
			log.Printf("Error copying to stream, read %d bytes from chunk: %s\n", n, err)
		}
	}
	stream.Close()
}

func storeRecipe(dest string, recipe []Chunk) {
	gob.Register(&StoredChunk{})
	gob.Register(&TempChunk{})
	gob.Register(&DeltaChunk{})
	file, err := os.Create(dest)
	if err == nil {
		encoder := gob.NewEncoder(file)
		for _, c := range recipe {
			if err = encoder.Encode(&c); err != nil {
				log.Panicln(err)
			}
		}
	}
	if err != nil {
		log.Panicln(err)
	}
	if err = file.Close(); err != nil {
		log.Panicln(err)
	}
}

func loadRecipe(path string) []Chunk {
	var recipe []Chunk
	gob.Register(&StoredChunk{})
	gob.Register(&TempChunk{})
	gob.Register(&DeltaChunk{})
	file, err := os.Open(path)
	if err == nil {
		decoder := gob.NewDecoder(file)
		for i := 0; err == nil; i++ {
			var c Chunk
			if err = decoder.Decode(&c); err == nil {
				recipe = append(recipe, c)
			}
		}
	}
	if err != nil && err != io.EOF {
		log.Panicln(err)
	}
	if err = file.Close(); err != nil {
		log.Panicln(err)
	}
	return recipe
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
