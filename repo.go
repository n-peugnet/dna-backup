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
│   ├── fingerprints
│   ├── recipe
│   └── sketches
└── 00001/
    ├── chunks/
    │   ├── 000000000000000
    │   ├── 000000000000001
    ├── files
│   ├── fingerprints
│   ├── recipe
│   └── sketches
```
*/

package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/chmduquesne/rollinghash/rabinkarp64"
	"github.com/n-peugnet/dna-backup/cache"
	"github.com/n-peugnet/dna-backup/logger"
	"github.com/n-peugnet/dna-backup/sketch"
	"github.com/n-peugnet/dna-backup/slice"
	"github.com/n-peugnet/dna-backup/utils"
)

func init() {
	// register chunk structs for encoding/decoding using gob
	gob.RegisterName("*dna-backup.StoredChunk", &StoredChunk{})
	gob.RegisterName("*dna-backup.TempChunk", &TempChunk{})
	gob.RegisterName("*dna-backup.DeltaChunk", &DeltaChunk{})
	gob.RegisterName("dna-backup.File", File{})
}

type FingerprintMap map[uint64]*ChunkId
type SketchMap map[uint64][]*ChunkId

func (m SketchMap) Set(key []uint64, value *ChunkId) {
	for _, s := range key {
		prev := m[s]
		if contains(prev, value) {
			continue
		}
		m[s] = append(prev, value)
	}
}

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
	recipe            []Chunk
	files             []File
	chunkCache        cache.Cacher
	chunkReadWrapper  utils.ReadWrapper
	chunkWriteWrapper utils.WriteWrapper
}

type chunkHashes struct {
	Fp uint64
	Sk []uint64
}

type chunkData struct {
	hashes  chunkHashes
	content []byte
	id      *ChunkId
}

type File struct {
	Path string
	Size int64
}

func NewRepo(path string) *Repo {
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		logger.Fatal(err)
	}
	err = os.MkdirAll(path, 0775)
	if err != nil {
		logger.Panic(err)
	}
	var seed int64 = 1
	p, err := rabinkarp64.RandomPolynomial(seed)
	if err != nil {
		logger.Panic(err)
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
		chunkCache:        cache.NewFifoCache(10000),
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
	source, err := filepath.Abs(source)
	if err != nil {
		logger.Fatal(err)
	}
	versions := r.loadVersions()
	newVersion := len(versions) // TODO: add newVersion functino
	newPath := filepath.Join(r.path, fmt.Sprintf(versionFmt, newVersion))
	newChunkPath := filepath.Join(newPath, chunksName)
	os.Mkdir(newPath, 0775)      // TODO: handle errors
	os.Mkdir(newChunkPath, 0775) // TODO: handle errors
	files := listFiles(source)
	r.loadHashes(versions)
	r.loadFileLists(versions)
	r.loadRecipes(versions)
	storeQueue := make(chan chunkData, 10)
	storeEnd := make(chan bool)
	go r.storageWorker(newVersion, storeQueue, storeEnd)
	var last, nlast, pass uint64
	var recipe []Chunk
	for ; nlast > last || pass == 0; pass++ {
		logger.Infof("pass number %d", pass+1)
		last = nlast
		reader, writer := io.Pipe()
		go concatFiles(&files, writer)
		recipe, nlast = r.matchStream(reader, storeQueue, newVersion, last)
	}
	close(storeQueue)
	<-storeEnd
	r.storeFileList(newVersion, unprefixFiles(files, source))
	r.storeRecipe(newVersion, recipe)
}

func (r *Repo) Restore(destination string) {
	versions := r.loadVersions()
	r.loadFileLists(versions)
	r.loadRecipes(versions)
	reader, writer := io.Pipe()
	go r.restoreStream(writer, r.recipe)
	bufReader := bufio.NewReaderSize(reader, r.chunkSize*2)
	for _, file := range r.files {
		filePath := filepath.Join(destination, file.Path)
		dir := filepath.Dir(filePath)
		os.MkdirAll(dir, 0775)      // TODO: handle errors
		f, _ := os.Create(filePath) // TODO: handle errors
		n, err := io.CopyN(f, bufReader, file.Size)
		if err != nil {
			logger.Errorf("storing file content for '%s', written %d/%d bytes: %s", filePath, n, file.Size, err)
		}
		if err := f.Close(); err != nil {
			logger.Errorf("closing restored file '%s': %s", filePath, err)
		}
	}
}

func (r *Repo) loadVersions() []string {
	var versions []string
	files, err := os.ReadDir(r.path)
	if err != nil {
		logger.Fatal(err)
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
	err := filepath.Walk(path, func(p string, i fs.FileInfo, err error) error {
		if err != nil {
			logger.Warning(err)
			return err
		}
		if i.IsDir() {
			return nil
		}
		files = append(files, File{p, i.Size()})
		return nil
	})
	if err != nil {
		// already logged in callback
	}
	return files
}

func unprefixFiles(files []File, prefix string) (ret []File) {
	var err error
	ret = make([]File, len(files))
	for i, f := range files {
		if f.Path, err = utils.Unprefix(f.Path, prefix); err != nil {
			logger.Warning(err)
		} else {
			ret[i] = f
		}
	}
	return
}

// concatFiles reads the content of all the listed files into a continuous stream.
// If any errors are encoutered while opening a file, it is then removed from the
// list.
// If read is incomplete, then the actual read size is used.
func concatFiles(files *[]File, stream io.WriteCloser) {
	actual := make([]File, 0, len(*files))
	for _, f := range *files {
		file, err := os.Open(f.Path)
		if err != nil {
			logger.Warning(err)
			continue
		}
		af := f
		if n, err := io.Copy(stream, file); err != nil {
			logger.Error("read ", n, " bytes, ", err)
			af.Size = n
		}
		actual = append(actual, af)
		if err = file.Close(); err != nil {
			logger.Panic(err)
		}
	}
	stream.Close()
	*files = actual
}

func storeBasicStruct(dest string, wrapper utils.WriteWrapper, obj interface{}) {
	file, err := os.Create(dest)
	if err != nil {
		logger.Panic(err)
	}
	out := wrapper(file)
	encoder := gob.NewEncoder(out)
	err = encoder.Encode(obj)
	if err != nil {
		logger.Panic(err)
	}
	if err = out.Close(); err != nil {
		logger.Panic(err)
	}
	if err = file.Close(); err != nil {
		logger.Panic(err)
	}
}

func loadBasicStruct(path string, wrapper utils.ReadWrapper, obj interface{}) {
	file, err := os.Open(path)
	if err != nil {
		logger.Panic(err)
	}
	in, err := wrapper(file)
	if err != nil {
		logger.Panic(err)
	}
	decoder := gob.NewDecoder(in)
	err = decoder.Decode(obj)
	if err != nil {
		logger.Panic(err)
	}
	if err = in.Close(); err != nil {
		logger.Panic(err)
	}
	if err = file.Close(); err != nil {
		logger.Panic(err)
	}
}

func (r *Repo) loadDeltas(versions []string, wrapper utils.ReadWrapper, name string) (ret slice.Slice) {
	for _, v := range versions {
		path := filepath.Join(v, name)
		var delta slice.Delta
		loadBasicStruct(path, wrapper, &delta)
		ret = slice.Patch(ret, delta)
	}
	return
}

func fileList2slice(l []File) (ret slice.Slice) {
	ret = make(slice.Slice, len(l))
	for i := range l {
		ret[i] = l[i]
	}
	return
}

func slice2fileList(s slice.Slice) (ret []File) {
	ret = make([]File, len(s), len(s))
	for i := range s {
		if f, ok := s[i].(File); ok {
			ret[i] = f
		} else {
			logger.Warningf("could not convert %s into a File", s[i])
		}
	}
	return
}

func (r *Repo) storeFileList(version int, list []File) {
	dest := filepath.Join(r.path, fmt.Sprintf(versionFmt, version), filesName)
	delta := slice.Diff(fileList2slice(r.files), fileList2slice(list))
	logger.Infof("files delta %s", delta.String())
	storeBasicStruct(dest, utils.NopWriteWrapper, delta)
}

func (r *Repo) loadFileLists(versions []string) {
	r.files = slice2fileList(r.loadDeltas(versions, utils.NopReadWrapper, filesName))
}

func (r *Repo) storageWorker(version int, storeQueue <-chan chunkData, end chan<- bool) {
	hashesFile := filepath.Join(r.path, fmt.Sprintf(versionFmt, version), hashesName)
	file, err := os.Create(hashesFile)
	if err != nil {
		logger.Panic(err)
	}
	encoder := gob.NewEncoder(file)
	for data := range storeQueue {
		err = encoder.Encode(data.hashes)
		err := r.StoreChunkContent(data.id, bytes.NewReader(data.content))
		if err != nil {
			logger.Error(err)
		}
		// logger.Debug("stored ", data.id)
	}
	if err = file.Close(); err != nil {
		logger.Panic(err)
	}
	end <- true
}

func (r *Repo) StoreChunkContent(id *ChunkId, reader io.Reader) error {
	path := id.Path(r.path)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating chunk for '%s'; %s\n", path, err)
	}
	wrapper := r.chunkWriteWrapper(file)
	n, err := io.Copy(wrapper, reader)
	if err != nil {
		return fmt.Errorf("writing chunk content for '%s', written %d bytes: %s\n", path, n, err)
	}
	if err := wrapper.Close(); err != nil {
		return fmt.Errorf("closing write wrapper for '%s': %s\n", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("closing chunk for '%s': %s\n", path, err)
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
			logger.Errorf("cannot open chunk '%s': %s", path, err)
		}
		wrapper, err := r.chunkReadWrapper(f)
		if err != nil {
			logger.Errorf("cannot create read wrapper for chunk '%s': %s", path, err)
		}
		value, err = io.ReadAll(wrapper)
		if err != nil {
			logger.Panicf("could not read from chunk '%s': %s", path, err)
		}
		if err = f.Close(); err != nil {
			logger.Warningf("could not close chunk '%s': %s", path, err)
		}
		r.chunkCache.Set(id, value)
	}
	return bytes.NewReader(value)
}

// TODO: use atoi for chunkid ?
func (r *Repo) loadChunks(versions []string, chunks chan<- IdentifiedChunk) {
	for i, v := range versions {
		p := filepath.Join(v, chunksName)
		entries, err := os.ReadDir(p)
		if err != nil {
			logger.Errorf("reading version '%05d' in '%s' chunks: %s", i, v, err)
		}
		for j, e := range entries {
			if e.IsDir() {
				continue
			}
			id := &ChunkId{Ver: i, Idx: uint64(j)}
			c := NewStoredChunk(r, id)
			chunks <- c
		}
	}
	close(chunks)
}

func (r *Repo) loadHashes(versions []string) {
	for i, v := range versions {
		path := filepath.Join(v, hashesName)
		file, err := os.Open(path)
		if err == nil {
			decoder := gob.NewDecoder(file)
			for j := 0; err == nil; j++ {
				var h chunkHashes
				if err = decoder.Decode(&h); err == nil {
					id := &ChunkId{i, uint64(j)}
					r.fingerprints[h.Fp] = id
					r.sketches.Set(h.Sk, id)
				}
			}
		}
		if err != nil && err != io.EOF {
			logger.Panic(err)
		}
		if err = file.Close(); err != nil {
			logger.Panic(err)
		}
	}
}

func (r *Repo) chunkMinLen() int {
	return sketch.SuperFeatureSize(r.chunkSize, r.sketchSfCount, r.sketchFCount)
}

// hashChunks calculates the hashes for a channel of chunks.
// For each chunk, both a fingerprint (hash over the full content) and a sketch
// (resemblance hash based on maximal values of regions) are calculated and
// stored in an hashmap.
func (r *Repo) hashChunks(chunks <-chan IdentifiedChunk) {
	for c := range chunks {
		r.hashChunk(c.GetId(), c.Reader())
	}
}

// hashChunk calculates the hashes for a chunk and store them in th repo hashmaps.
func (r *Repo) hashChunk(id *ChunkId, reader io.Reader) (fp uint64, sk []uint64) {
	var buffSk bytes.Buffer
	var buffFp bytes.Buffer
	var wg sync.WaitGroup
	reader = io.TeeReader(reader, &buffSk)
	io.Copy(&buffFp, reader)
	wg.Add(2)
	go r.makeFingerprint(id, &buffFp, &wg, &fp)
	go r.makeSketch(id, &buffSk, &wg, &sk)
	wg.Wait()
	if _, e := r.fingerprints[fp]; e {
		logger.Error(fp, " already exists in fingerprints map")
	}
	r.fingerprints[fp] = id
	r.sketches.Set(sk, id)
	return
}

func (r *Repo) makeFingerprint(id *ChunkId, reader io.Reader, wg *sync.WaitGroup, ret *uint64) {
	defer wg.Done()
	hasher := rabinkarp64.NewFromPol(r.pol)
	io.Copy(hasher, reader)
	*ret = hasher.Sum64()
}

func (r *Repo) makeSketch(id *ChunkId, reader io.Reader, wg *sync.WaitGroup, ret *[]uint64) {
	defer wg.Done()
	*ret, _ = sketch.SketchChunk(reader, r.pol, r.chunkSize, r.sketchWSize, r.sketchSfCount, r.sketchFCount)
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
			logger.Debugf("found %d %d time(s)", id, count)
			if count > max {
				max = count
				similarChunk = id
			}
			similarChunks[*id] = count
		}
	}
	return similarChunk, max > 0
}

func (r *Repo) tryDeltaEncodeChunk(temp BufferedChunk) (Chunk, bool) {
	id, found := r.findSimilarChunk(temp)
	if found {
		var buff bytes.Buffer
		if err := r.differ.Diff(r.LoadChunkContent(id), temp.Reader(), &buff); err != nil {
			logger.Error("trying delta encode chunk:", temp, "with source:", id, ":", err)
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
func (r *Repo) encodeTempChunk(temp BufferedChunk, version int, last *uint64, storeQueue chan<- chunkData) (chunk Chunk, isDelta bool) {
	chunk, isDelta = r.tryDeltaEncodeChunk(temp)
	if isDelta {
		logger.Debug("add new delta chunk")
		return
	}
	if chunk.Len() == r.chunkSize {
		id := &ChunkId{Ver: version, Idx: *last}
		*last++
		fp, sk := r.hashChunk(id, temp.Reader())
		storeQueue <- chunkData{
			hashes:  chunkHashes{fp, sk},
			content: temp.Bytes(),
			id:      id,
		}
		r.chunkCache.Set(id, temp.Bytes())
		logger.Debug("add new chunk ", id)
		return NewStoredChunk(r, id), false
	}
	logger.Debug("add new partial chunk of size: ", chunk.Len())
	return
}

// encodeTempChunks encodes the current temporary chunks based on the value of the previous one.
// Temporary chunks can be partial. If the current chunk is smaller than the size of a
// super-feature and there exists a previous chunk, then both are merged before attempting
// to delta-encode them.
func (r *Repo) encodeTempChunks(prev BufferedChunk, curr BufferedChunk, version int, last *uint64, storeQueue chan<- chunkData) []Chunk {
	if reflect.ValueOf(prev).IsNil() {
		c, _ := r.encodeTempChunk(curr, version, last, storeQueue)
		return []Chunk{c}
	} else if curr.Len() < r.chunkMinLen() {
		tmp := NewTempChunk(append(prev.Bytes(), curr.Bytes()...))
		c, success := r.encodeTempChunk(tmp, version, last, storeQueue)
		if success {
			return []Chunk{c}
		}
	}
	prevD, _ := r.encodeTempChunk(prev, version, last, storeQueue)
	currD, _ := r.encodeTempChunk(curr, version, last, storeQueue)
	return []Chunk{prevD, currD}
}

func (r *Repo) matchStream(stream io.Reader, storeQueue chan<- chunkData, version int, last uint64) ([]Chunk, uint64) {
	var b byte
	var chunks []Chunk
	var prev *TempChunk
	var err error
	bufStream := bufio.NewReaderSize(stream, r.chunkSize*2)
	buff := make([]byte, r.chunkSize, r.chunkSize*2)
	if n, err := io.ReadFull(stream, buff); n < r.chunkSize {
		if err == io.ErrUnexpectedEOF {
			c, _ := r.encodeTempChunk(NewTempChunk(buff[:n]), version, &last, storeQueue)
			chunks = append(chunks, c)
			return chunks, last
		} else {
			logger.Panicf("matching stream, read only %d bytes with error '%s'", n, err)
		}
	}
	hasher := rabinkarp64.NewFromPol(r.pol)
	hasher.Write(buff)
	for err != io.EOF {
		h := hasher.Sum64()
		chunkId, exists := r.fingerprints[h]
		if exists {
			if len(buff) > r.chunkSize && len(buff) <= r.chunkSize*2 {
				size := len(buff) - r.chunkSize
				temp := NewTempChunk(buff[:size])
				chunks = append(chunks, r.encodeTempChunks(prev, temp, version, &last, storeQueue)...)
				prev = nil
			} else if prev != nil {
				c, _ := r.encodeTempChunk(prev, version, &last, storeQueue)
				chunks = append(chunks, c)
				prev = nil
			}
			logger.Debugf("add existing chunk: %d", chunkId)
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
				chunk, _ := r.encodeTempChunk(prev, version, &last, storeQueue)
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
				chunk, _ := r.encodeTempChunk(prev, version, &last, storeQueue)
				chunks = append(chunks, chunk)
			}
			prev = NewTempChunk(buff[:r.chunkSize])
			temp = NewTempChunk(buff[r.chunkSize:])
		} else {
			temp = NewTempChunk(buff)
		}
		chunks = append(chunks, r.encodeTempChunks(prev, temp, version, &last, storeQueue)...)
	}
	return chunks, last
}

func (r *Repo) restoreStream(stream io.WriteCloser, recipe []Chunk) {
	for _, c := range recipe {
		if n, err := io.Copy(stream, c.Reader()); err != nil {
			logger.Errorf("copying to stream, read %d bytes from chunk: %s", n, err)
		}
	}
	stream.Close()
}

func recipe2slice(r []Chunk) (ret slice.Slice) {
	ret = make(slice.Slice, len(r))
	for i := range r {
		ret[i] = r[i]
	}
	return
}

func slice2recipe(s slice.Slice) (ret []Chunk) {
	ret = make([]Chunk, len(s), len(s))
	for i := range s {
		if c, ok := s[i].(Chunk); ok {
			ret[i] = c
		} else {
			logger.Warningf("could not convert %s into a Chunk", s[i])
		}
	}
	return
}

func (r *Repo) storeRecipe(version int, recipe []Chunk) {
	dest := filepath.Join(r.path, fmt.Sprintf(versionFmt, version), recipeName)
	delta := slice.Diff(recipe2slice(r.recipe), recipe2slice(recipe))
	logger.Infof("recipe delta %s", delta.String())
	storeBasicStruct(dest, utils.NopWriteWrapper, delta)
}

func (r *Repo) loadRecipes(versions []string) {
	recipe := slice2recipe(r.loadDeltas(versions, utils.NopReadWrapper, recipeName))
	for _, c := range recipe {
		if rc, isRepo := c.(RepoChunk); isRepo {
			rc.SetRepo(r)
		}
	}
	r.recipe = recipe
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
