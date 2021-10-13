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
│   │   └── 000000000000003
│   ├── files
│   ├── hashes
│   └── recipe
└── 00001/
    ├── chunks/
    │   ├── 000000000000000
    │   └── 000000000000001
    ├── files
    ├── hashes
    └── recipe
```
*/

package repo

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
	"strings"
	"sync"

	"github.com/chmduquesne/rollinghash/rabinkarp64"
	"github.com/n-peugnet/dna-backup/cache"
	"github.com/n-peugnet/dna-backup/delta"
	"github.com/n-peugnet/dna-backup/logger"
	"github.com/n-peugnet/dna-backup/sketch"
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
	versions          []string
	chunkSize         int
	sketchWSize       int
	sketchSfCount     int
	sketchFCount      int
	pol               rabinkarp64.Pol
	differ            delta.Differ
	patcher           delta.Patcher
	fingerprints      FingerprintMap
	sketches          SketchMap
	recipe            []Chunk
	recipeRaw         []byte
	files             []File
	filesRaw          []byte
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
	Link string
}

func NewRepo(path string, chunkSize int) *Repo {
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
		chunkSize:         chunkSize,
		sketchWSize:       32,
		sketchSfCount:     3,
		sketchFCount:      4,
		pol:               p,
		differ:            delta.Fdelta{},
		patcher:           delta.Fdelta{},
		fingerprints:      make(FingerprintMap),
		sketches:          make(SketchMap),
		chunkCache:        cache.NewFifoCache(10000),
		chunkReadWrapper:  utils.ZlibReader,
		chunkWriteWrapper: utils.ZlibWriter,
	}
}

func (r *Repo) Differ() delta.Differ {
	return r.differ
}

func (r *Repo) Patcher() delta.Patcher {
	return r.patcher
}

func (r *Repo) Commit(source string) {
	source, err := filepath.Abs(source)
	if err != nil {
		logger.Fatal(err)
	}
	r.Init()
	newVersion := len(r.versions) // TODO: add newVersion functino
	newPath := filepath.Join(r.path, fmt.Sprintf(versionFmt, newVersion))
	newChunkPath := filepath.Join(newPath, chunksName)
	os.Mkdir(newPath, 0775)      // TODO: handle errors
	os.Mkdir(newChunkPath, 0775) // TODO: handle errors
	files := listFiles(source)
	storeQueue := make(chan chunkData, 32)
	storeEnd := make(chan bool)
	go r.storageWorker(newVersion, storeQueue, storeEnd)
	var last, nlast, pass uint64
	var recipe []Chunk
	for ; nlast > last || pass == 0; pass++ {
		logger.Infof("matcher pass number %d", pass+1)
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
	r.Init()
	reader, writer := io.Pipe()
	logger.Info("restore latest version")
	go r.restoreStream(writer, r.recipe)
	bufReader := bufio.NewReaderSize(reader, r.chunkSize*2)
	for _, file := range r.files {
		filePath := filepath.Join(destination, file.Path)
		dir := filepath.Dir(filePath)
		os.MkdirAll(dir, 0775) // TODO: handle errors
		if file.Link != "" {
			link := file.Link
			if filepath.IsAbs(link) {
				filepath.Join(destination, file.Link)
			}
			err := os.Symlink(link, filePath)
			if err != nil {
				logger.Errorf("restored symlink ", err)
			}
		} else {
			f, _ := os.Create(filePath) // TODO: handle errors
			n, err := io.CopyN(f, bufReader, file.Size)
			if err != nil {
				logger.Errorf("restored file, written %d/%d bytes: %s", filePath, n, file.Size, err)
			}
			if err := f.Close(); err != nil {
				logger.Errorf("restored file ", err)
			}
		}
	}
}

func (r *Repo) Init() {
	var wg sync.WaitGroup
	r.loadVersions()
	wg.Add(3)
	go r.loadHashes(r.versions, &wg)
	go r.loadFileLists(r.versions, &wg)
	go r.loadRecipes(r.versions, &wg)
	wg.Wait()
}

func (r *Repo) loadVersions() {
	files, err := os.ReadDir(r.path)
	if err != nil {
		logger.Fatal(err)
	}
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		r.versions = append(r.versions, filepath.Join(r.path, f.Name()))
	}
}

func listFiles(path string) []File {
	logger.Infof("list files from %s", path)
	var files []File
	err := filepath.Walk(path, func(p string, i fs.FileInfo, err error) error {
		if err != nil {
			logger.Warning(err)
			return nil
		}
		if i.IsDir() {
			return nil
		}
		var file = File{Path: p, Size: i.Size()}
		if i.Mode()&fs.ModeSymlink != 0 {
			file, err = cleanSymlink(path, p, i)
			if err != nil {
				logger.Warning("skipping symlink ", err)
				return nil
			}
		}
		files = append(files, file)
		return nil
	})
	if err != nil {
		logger.Error(err)
	}
	return files
}

func cleanSymlink(root string, p string, i fs.FileInfo) (f File, err error) {
	dir := filepath.Dir(p)
	target, err := os.Readlink(p)
	if err != nil {
		return
	}
	isAbs := filepath.IsAbs(target)
	cleaned := target
	if !isAbs {
		cleaned = filepath.Join(dir, cleaned)
	}
	cleaned = filepath.Clean(cleaned)
	if !strings.HasPrefix(cleaned, root) {
		err = fmt.Errorf("external %s -> %s", p, cleaned)
		return
	}
	if isAbs {
		f.Link, err = utils.Unprefix(cleaned, root)
	} else {
		f.Link, err = filepath.Rel(dir, filepath.Join(dir, target))
	}
	if err != nil {
		return
	}
	if f.Link == "" {
		err = fmt.Errorf("empty %s", p)
		return
	}
	f.Path = p
	f.Size = 0
	return f, nil
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
//
// If read is incomplete, then the actual read size is used.
func concatFiles(files *[]File, stream io.WriteCloser) {
	actual := make([]File, 0, len(*files))
	for _, f := range *files {
		if f.Link != "" {
			actual = append(actual, f)
			continue
		}
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

func storeDelta(prevRaw []byte, curr interface{}, dest string, differ delta.Differ, wrapper utils.WriteWrapper) {
	var prevBuff, currBuff bytes.Buffer
	var encoder *gob.Encoder
	var err error

	prevBuff = *bytes.NewBuffer(prevRaw)
	encoder = gob.NewEncoder(&currBuff)
	if err = encoder.Encode(curr); err != nil {
		logger.Panic(err)
	}
	logger.Infof("store before delta: %d", currBuff.Len())
	file, err := os.Create(dest)
	if err != nil {
		logger.Panic(err)
	}
	out := wrapper(file)
	if err = differ.Diff(&prevBuff, &currBuff, out); err != nil {
		logger.Panic(err)
	}
	if err = out.Close(); err != nil {
		logger.Panic(err)
	}
	if err = file.Close(); err != nil {
		logger.Panic(err)
	}
}

func loadDeltas(target interface{}, versions []string, patcher delta.Patcher, wrapper utils.ReadWrapper, name string) (ret []byte) {
	var prev bytes.Buffer
	var err error

	for _, v := range versions {
		var curr bytes.Buffer
		path := filepath.Join(v, name)
		file, err := os.Open(path)
		if err != nil {
			logger.Panic(err)
		}
		in, err := wrapper(file)
		if err != nil {
			logger.Panic(err)
		}
		if err = patcher.Patch(&prev, &curr, in); err != nil {
			logger.Panic(err)
		}
		prev = curr
		if err = in.Close(); err != nil {
			logger.Panic(err)
		}
		if err = file.Close(); err != nil {
			logger.Panic(err)
		}
	}
	ret = prev.Bytes()
	if len(ret) == 0 {
		return
	}
	decoder := gob.NewDecoder(&prev)
	if err = decoder.Decode(target); err != nil {
		logger.Panic(err)
	}
	return
}

// storeFileList stores the given list in the repo dir as a delta against the
// previous version's one.
func (r *Repo) storeFileList(version int, list []File) {
	logger.Info("store files")
	dest := filepath.Join(r.path, fmt.Sprintf(versionFmt, version), filesName)
	storeDelta(r.filesRaw, list, dest, r.differ, r.chunkWriteWrapper)
}

// loadFileLists loads incrementally the file lists' delta of each given version.
func (r *Repo) loadFileLists(versions []string, wg *sync.WaitGroup) {
	logger.Info("load previous file lists")
	var files []File
	r.filesRaw = loadDeltas(&files, versions, r.patcher, r.chunkReadWrapper, filesName)
	r.files = files
	wg.Done()
}

// storageWorker is meant to be started in a goroutine and stores each new chunk's
// data in the repo directory until the store queue channel is closed.
//
// it will put true in the end channel once everything is stored.
func (r *Repo) storageWorker(version int, storeQueue <-chan chunkData, end chan<- bool) {
	hashesFile := filepath.Join(r.path, fmt.Sprintf(versionFmt, version), hashesName)
	file, err := os.Create(hashesFile)
	if err != nil {
		logger.Panic(err)
	}
	encoder := gob.NewEncoder(file)
	for data := range storeQueue {
		err = encoder.Encode(data.hashes)
		r.StoreChunkContent(data.id, bytes.NewReader(data.content))
		// logger.Debug("stored ", data.id)
	}
	if err = file.Close(); err != nil {
		logger.Panic(err)
	}
	end <- true
}

func (r *Repo) StoreChunkContent(id *ChunkId, reader io.Reader) {
	path := id.Path(r.path)
	file, err := os.Create(path)
	if err != nil {
		logger.Panic("chunk store ", err)
	}
	wrapper := r.chunkWriteWrapper(file)
	n, err := io.Copy(wrapper, reader)
	if err != nil {
		logger.Errorf("chunk store, %d written, %s", n, err)
	}
	if err := wrapper.Close(); err != nil {
		logger.Warning("chunk store wrapper ", err)
	}
	if err := file.Close(); err != nil {
		logger.Warning("chunk store ", err)
	}
}

// LoadChunkContent loads a chunk from the repo directory.
// If the chunk is in cache, get it from cache, else read it from drive.
func (r *Repo) LoadChunkContent(id *ChunkId) *bytes.Reader {
	value, exists := r.chunkCache.Get(id)
	if !exists {
		path := id.Path(r.path)
		f, err := os.Open(path)
		if err != nil {
			logger.Panic("chunk load ", err)
		}
		wrapper, err := r.chunkReadWrapper(f)
		if err != nil {
			logger.Error("chunk load wrapper ", err)
		}
		value, err = io.ReadAll(wrapper)
		if err != nil {
			logger.Error("chunk load ", err)
		}
		if err = wrapper.Close(); err != nil {
			logger.Warning("chunk load wrapper", err)
		}
		if err = f.Close(); err != nil {
			logger.Warning("chunk load ", err)
		}
		r.chunkCache.Set(id, value)
	}
	return bytes.NewReader(value)
}

// TODO: use atoi for chunkid ?
func (r *Repo) LoadChunks(chunks chan<- IdentifiedChunk) {
	for i, v := range r.versions {
		p := filepath.Join(v, chunksName)
		entries, err := os.ReadDir(p)
		if err != nil {
			logger.Error("version dir ", err)
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

// loadHashes loads and aggregates the hashes stored for each given version and
// stores them in the repo maps.
func (r *Repo) loadHashes(versions []string, wg *sync.WaitGroup) {
	logger.Info("load previous hashes")
	for i, v := range versions {
		path := filepath.Join(v, hashesName)
		file, err := os.Open(path)
		if err != nil {
			logger.Error("hashes ", err)
		}
		decoder := gob.NewDecoder(file)
		for j := 0; err == nil; j++ {
			var h chunkHashes
			if err = decoder.Decode(&h); err == nil {
				id := &ChunkId{i, uint64(j)}
				r.fingerprints[h.Fp] = id
				r.sketches.Set(h.Sk, id)
			}
		}
		if err != nil && err != io.EOF {
			logger.Panic(err)
		}
		if err = file.Close(); err != nil {
			logger.Warning(err)
		}
	}
	wg.Done()
}

func (r *Repo) chunkMinLen() int {
	return sketch.SuperFeatureSize(r.chunkSize, r.sketchSfCount, r.sketchFCount)
}

func contains(s []*ChunkId, id *ChunkId) bool {
	for _, v := range s {
		if v == id {
			return true
		}
	}
	return false
}

// findSimilarChunk looks in the repo sketch map for a match of the given sketch.
//
// There can be multiple matches but only the best one is returned. Indeed, the
// more superfeature matches, the better the quality of the match. For now we
// consider that a single superfeature match is enough to count it as valid.
func (r *Repo) findSimilarChunk(sketch []uint64) (*ChunkId, bool) {
	var similarChunks = make(map[ChunkId]int)
	var max int
	var similarChunk *ChunkId
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

// encodeTempChunk first tries to delta-encode the given chunk before attributing
// it an Id and saving it into the fingerprints and sketches maps.
func (r *Repo) encodeTempChunk(temp BufferedChunk, version int, last *uint64, storeQueue chan<- chunkData) (Chunk, bool) {
	sk, _ := sketch.SketchChunk(temp.Reader(), r.pol, r.chunkSize, r.sketchWSize, r.sketchSfCount, r.sketchFCount)
	id, found := r.findSimilarChunk(sk)
	if found {
		var buff bytes.Buffer
		if err := r.differ.Diff(r.LoadChunkContent(id), temp.Reader(), &buff); err != nil {
			logger.Error("trying delta encode chunk:", temp, "with source:", id, ":", err)
		} else {
			logger.Debugf("add new delta chunk of size %d", len(buff.Bytes()))
			return &DeltaChunk{
				repo:   r,
				Source: id,
				Patch:  buff.Bytes(),
				Size:   temp.Len(),
			}, true
		}
	}
	if temp.Len() == r.chunkSize {
		id := &ChunkId{Ver: version, Idx: *last}
		*last++
		hasher := rabinkarp64.NewFromPol(r.pol)
		io.Copy(hasher, temp.Reader())
		fp := hasher.Sum64()
		r.fingerprints[fp] = id
		r.sketches.Set(sk, id)
		storeQueue <- chunkData{
			hashes:  chunkHashes{fp, sk},
			content: temp.Bytes(),
			id:      id,
		}
		r.chunkCache.Set(id, temp.Bytes())
		logger.Debug("add new chunk ", id)
		return NewStoredChunk(r, id), false
	}
	logger.Debug("add new partial chunk of size: ", temp.Len())
	return temp, false
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

// matchStream is the heart of DNA-backup. Thus, it sounded rude not to add some comment to it.
//
// It applies a rolling hash on the content of a given stream to look for matching fingerprints
// in the repo. If no match is found after the equivalent of three chunks of data are processed,
// then the first unmatched chunk sketch is checked to see if it could be delta-encoded.
// If not, the chunk is then stored as a new chunk for this version and its fingerprint and
// sketch are added to the repo maps.
//
// If a match happens during the processing of the third chunk, then, if possible, the remaining
// of the second chunk is merged with the first one to try to delta encode it at once.
//
// Each time a new chunk is added it is sent to the store worker through the store queue.
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

func (r *Repo) storeRecipe(version int, recipe []Chunk) {
	logger.Info("store recipe")
	dest := filepath.Join(r.path, fmt.Sprintf(versionFmt, version), recipeName)
	storeDelta(r.recipeRaw, recipe, dest, r.differ, r.chunkWriteWrapper)
}

func (r *Repo) loadRecipes(versions []string, wg *sync.WaitGroup) {
	logger.Info("load previous recipies")
	var recipe []Chunk
	r.recipeRaw = loadDeltas(&recipe, versions, r.patcher, r.chunkReadWrapper, recipeName)
	for _, c := range recipe {
		if rc, isRepo := c.(RepoChunk); isRepo {
			rc.SetRepo(r)
		}
	}
	r.recipe = recipe
	wg.Done()
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
