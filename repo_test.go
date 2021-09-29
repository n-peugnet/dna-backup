package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/chmduquesne/rollinghash/rabinkarp64"
	"github.com/n-peugnet/dna-backup/logger"
	"github.com/n-peugnet/dna-backup/sketch"
	"github.com/n-peugnet/dna-backup/testutils"
	"github.com/n-peugnet/dna-backup/utils"
)

func chunkCompare(t *testing.T, dataDir string, repo *Repo, testFiles []string, chunkCount int) {
	reader, writer := io.Pipe()
	chunks := make(chan []byte)
	files := listFiles(dataDir)
	go concatFiles(&files, writer)
	go repo.chunkStream(reader, chunks)

	offset := 0
	buff := make([]byte, repo.chunkSize*chunkCount)
	for _, f := range testFiles {
		content, err := os.ReadFile(filepath.Join(dataDir, f))
		if err != nil {
			t.Error("Error reading test data file")
		}
		for i := range content {
			buff[offset+i] = content[i]
		}
		offset += len(content)
	}

	i := 0
	for c := range chunks {
		start := i * repo.chunkSize
		end := (i + 1) * repo.chunkSize
		if end > offset {
			end = offset
		}
		content := buff[start:end]
		if bytes.Compare(c, content) != 0 {
			t.Errorf("Chunk %d does not match file content", i)
			// for i, b := range c {
			// 	fmt.Printf("E: %d, A: %d\n", b, content[i])
			// }
			t.Log("Expected: ", c[:10], "...", c[end%repo.chunkSize-10:])
			t.Log("Actual:", content)
		}
		i++
	}
	if i != chunkCount {
		t.Errorf("Incorrect number of chunks: %d, should be: %d", i, chunkCount)
	}
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
			logger.Error(err)
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

func storeChunks(dest string, chunks <-chan []byte) {
	i := 0
	for c := range chunks {
		path := filepath.Join(dest, fmt.Sprintf(chunkIdFmt, i))
		err := os.WriteFile(path, c, 0664)
		if err != nil {
			logger.Error(err)
		}
		i++
	}
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

func TestReadFiles1(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewRepo(tmpDir)
	chunkCount := 590/repo.chunkSize + 1
	dataDir := filepath.Join("testdata", "logs", "1")
	files := []string{"logTest.log"}
	chunkCompare(t, dataDir, repo, files, chunkCount)
}

func TestReadFiles2(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewRepo(tmpDir)
	chunkCount := 22899/repo.chunkSize + 1
	dataDir := filepath.Join("testdata", "logs", "2")
	files := []string{"csvParserTest.log", "slipdb.log"}
	chunkCompare(t, dataDir, repo, files, chunkCount)
}

func TestReadFiles3(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewRepo(tmpDir)
	chunkCount := 119398/repo.chunkSize + 1
	dataDir := filepath.Join("testdata", "logs")
	files := []string{
		filepath.Join("1", "logTest.log"),
		filepath.Join("2", "csvParserTest.log"),
		filepath.Join("2", "slipdb.log"),
		filepath.Join("3", "indexingTreeTest.log"),
	}
	chunkCompare(t, dataDir, repo, files, chunkCount)
}

func TestSymlinks(t *testing.T) {
	var output bytes.Buffer
	logger.SetOutput(&output)
	defer logger.SetOutput(os.Stderr)
	tmpDir := t.TempDir()
	extDir := t.TempDir()
	f, err := os.Create(filepath.Join(tmpDir, "existing"))
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	os.Symlink(extDir, filepath.Join(tmpDir, "linkexternal"))
	os.Symlink("./notexisting", filepath.Join(tmpDir, "linknotexisting"))
	os.Symlink("./existing", filepath.Join(tmpDir, "linkexisting"))
	files := listFiles(tmpDir)
	testutils.AssertLen(t, 2, files, "Files")
	if files[0].Link != "" {
		t.Error("linkexisting should not be a link, actual:", files[0].Link)
	}
	if files[1].Link != "existing" {
		t.Error("linkexisting should point to 'existing', actual:", files[1].Link)
	}
	if !strings.Contains(output.String(), "linkexternal") {
		t.Errorf("log should contain a warning for linkexternal, actual %q", &output)
	}
	if !strings.Contains(output.String(), "notexisting") {
		t.Errorf("log should contain a warning for notexisting, actual %q", &output)
	}
}

func TestLoadChunks(t *testing.T) {
	resultDir := t.TempDir()
	dataDir := filepath.Join("testdata", "logs")
	repo := NewRepo(resultDir)
	repo.chunkReadWrapper = utils.NopReadWrapper
	repo.chunkWriteWrapper = utils.NopWriteWrapper
	resultVersion := filepath.Join(resultDir, "00000")
	resultChunks := filepath.Join(resultVersion, chunksName)
	os.MkdirAll(resultChunks, 0775)
	reader1, writer1 := io.Pipe()
	reader2, writer2 := io.Pipe()
	chunks1 := make(chan []byte, 16)
	chunks2 := make(chan []byte, 16)
	chunks3 := make(chan IdentifiedChunk, 16)
	files := listFiles(dataDir)
	go concatFiles(&files, writer1)
	go concatFiles(&files, writer2)
	go repo.chunkStream(reader1, chunks1)
	go repo.chunkStream(reader2, chunks2)
	storeChunks(resultChunks, chunks1)
	versions := []string{resultVersion}
	go repo.loadChunks(versions, chunks3)

	i := 0
	for c2 := range chunks2 {
		c3 := <-chunks3
		buff, err := io.ReadAll(c3.Reader())
		if err != nil {
			t.Errorf("Error reading from chunk %d: %s\n", c3, err)
		}
		if bytes.Compare(c2, buff) != 0 {
			t.Errorf("Chunk %d does not match file content", i)
			t.Log("Expected: ", c2[:10], "...")
			t.Log("Actual:", buff)
		}
		i++
	}
}

func prepareChunks(dataDir string, repo *Repo, streamFunc func(*[]File, io.WriteCloser)) {
	resultVersion := filepath.Join(repo.path, "00000")
	resultChunks := filepath.Join(resultVersion, chunksName)
	os.MkdirAll(resultChunks, 0775)
	reader := getDataStream(dataDir, streamFunc)
	chunks := make(chan []byte, 16)
	go repo.chunkStream(reader, chunks)
	storeChunks(resultChunks, chunks)
}

func getDataStream(dataDir string, streamFunc func(*[]File, io.WriteCloser)) io.Reader {
	reader, writer := io.Pipe()
	files := listFiles(dataDir)
	go streamFunc(&files, writer)
	return reader
}

func TestBsdiff(t *testing.T) {
	logger.SetLevel(3)
	defer logger.SetLevel(4)
	resultDir := t.TempDir()
	repo := NewRepo(resultDir)
	dataDir := filepath.Join("testdata", "logs")
	addedFile1 := filepath.Join(dataDir, "2", "slogTest.log")
	addedFile2 := filepath.Join(dataDir, "3", "slogTest.log")
	// Store initial chunks
	prepareChunks(dataDir, repo, concatFiles)

	// Modify data
	ioutil.WriteFile(addedFile1, []byte("hello"), 0664)
	defer os.Remove(addedFile1)
	ioutil.WriteFile(addedFile2, make([]byte, 4000), 0664)
	defer os.Remove(addedFile2)

	// configure repo
	repo.patcher = Bsdiff{}
	repo.differ = Bsdiff{}
	repo.chunkReadWrapper = utils.NopReadWrapper
	repo.chunkWriteWrapper = utils.NopWriteWrapper

	// Load previously stored chunks
	oldChunks := make(chan IdentifiedChunk, 16)
	versions := repo.loadVersions()
	go repo.loadChunks(versions, oldChunks)
	repo.hashChunks(oldChunks)

	// Read new data
	newVersion := len(versions)
	newPath := filepath.Join(repo.path, fmt.Sprintf(versionFmt, newVersion))
	os.MkdirAll(newPath, 0775)
	reader := getDataStream(dataDir, concatFiles)
	storeQueue := make(chan chunkData, 10)
	storeEnd := make(chan bool)
	go repo.storageWorker(newVersion, storeQueue, storeEnd)
	recipe, _ := repo.matchStream(reader, storeQueue, newVersion, 0)
	close(storeQueue)
	<-storeEnd
	newChunks := extractDeltaChunks(recipe)
	testutils.AssertLen(t, 2, newChunks, "New delta chunks:")
	for _, c := range newChunks {
		logger.Info("Patch size:", len(c.Patch))
		if len(c.Patch) >= repo.chunkSize/10 {
			t.Errorf("Bsdiff of chunk is too large: %d", len(c.Patch))
		}
	}
}

func TestCommit(t *testing.T) {
	dest := t.TempDir()
	source := filepath.Join("testdata", "logs")
	expected := filepath.Join("testdata", "repo_8k")
	repo := NewRepo(dest)
	repo.patcher = Bsdiff{}
	repo.differ = Bsdiff{}
	repo.chunkReadWrapper = utils.NopReadWrapper
	repo.chunkWriteWrapper = utils.NopWriteWrapper

	repo.Commit(source)
	assertSameTree(t, assertCompatibleRepoFile, expected, dest, "Commit")
}

func TestCommitZlib(t *testing.T) {
	dest := t.TempDir()
	source := filepath.Join("testdata", "logs")
	expected := filepath.Join("testdata", "repo_8k_zlib")
	repo := NewRepo(dest)
	repo.patcher = Bsdiff{}
	repo.differ = Bsdiff{}
	repo.chunkReadWrapper = utils.ZlibReader
	repo.chunkWriteWrapper = utils.ZlibWriter

	repo.Commit(source)
	assertSameTree(t, assertCompatibleRepoFile, expected, dest, "Commit")
}

func TestRestore(t *testing.T) {
	logger.SetLevel(2)
	defer logger.SetLevel(4)
	dest := t.TempDir()
	source := filepath.Join("testdata", "repo_8k")
	expected := filepath.Join("testdata", "logs")
	repo := NewRepo(source)
	repo.patcher = Bsdiff{}
	repo.differ = Bsdiff{}
	repo.chunkReadWrapper = utils.NopReadWrapper
	repo.chunkWriteWrapper = utils.NopWriteWrapper

	repo.Restore(dest)
	assertSameTree(t, testutils.AssertSameFile, expected, dest, "Restore")
}

func TestRestoreZlib(t *testing.T) {
	logger.SetLevel(2)
	defer logger.SetLevel(4)
	dest := t.TempDir()
	source := filepath.Join("testdata", "repo_8k_zlib")
	expected := filepath.Join("testdata", "logs")
	repo := NewRepo(source)
	repo.patcher = Bsdiff{}
	repo.differ = Bsdiff{}
	repo.chunkReadWrapper = utils.ZlibReader
	repo.chunkWriteWrapper = utils.ZlibWriter

	repo.Restore(dest)
	assertSameTree(t, testutils.AssertSameFile, expected, dest, "Restore")
}

func TestRoundtrip(t *testing.T) {
	logger.SetLevel(2)
	defer logger.SetLevel(4)
	temp := t.TempDir()
	dest := t.TempDir()
	source := filepath.Join("testdata", "logs")
	repo1 := NewRepo(temp)
	repo2 := NewRepo(temp)

	repo1.Commit(source)
	// Commit a second version, just to see if it does not destroy everything
	// TODO: check that the second version is indeed empty
	repo1.Commit(source)
	repo2.Restore(dest)

	assertSameTree(t, assertCompatibleRepoFile, source, dest, "Commit")
}

func TestHashes(t *testing.T) {
	dest := t.TempDir()
	source := filepath.Join("testdata", "repo_8k")

	chunks := make(chan IdentifiedChunk, 16)
	storeQueue := make(chan chunkData, 16)
	storeEnd := make(chan bool)

	repo1 := NewRepo(source)
	repo1.chunkReadWrapper = utils.NopReadWrapper
	repo1.chunkWriteWrapper = utils.NopWriteWrapper
	go repo1.loadChunks([]string{filepath.Join(source, "00000")}, chunks)
	for c := range chunks {
		fp, sk := repo1.hashChunk(c.GetId(), c.Reader())
		content, err := io.ReadAll(c.Reader())
		if err != nil {
			t.Error(err)
		}
		storeQueue <- chunkData{
			hashes:  chunkHashes{fp, sk},
			content: content,
			id:      c.GetId(),
		}
	}
	repo2 := NewRepo(dest)
	repo2.chunkReadWrapper = utils.NopReadWrapper
	repo2.chunkWriteWrapper = utils.NopWriteWrapper
	os.MkdirAll(filepath.Join(dest, "00000", chunksName), 0775)
	go repo2.storageWorker(0, storeQueue, storeEnd)
	close(storeQueue)
	<-storeEnd
	testutils.AssertLen(t, 0, repo2.fingerprints, "Fingerprints")
	testutils.AssertLen(t, 0, repo2.sketches, "Sketches")
	repo2.loadHashes([]string{filepath.Join(dest, "00000")})
	testutils.AssertSame(t, repo1.fingerprints, repo2.fingerprints, "Fingerprint maps")
	testutils.AssertSame(t, repo1.sketches, repo2.sketches, "Sketches maps")
}

func assertSameTree(t *testing.T, apply func(t *testing.T, expected string, actual string, prefix string), expected string, actual string, prefix string) {
	actualFiles := listFiles(actual)
	expectedFiles := listFiles(expected)
	efCount := len(expectedFiles)
	if efCount <= 0 {
		t.Fatalf("No expected files: %d", efCount)
	}
	afCount := len(actualFiles)
	if efCount != afCount {
		t.Fatalf("Incorrect number of files: %d, should be %d", afCount, efCount)
	}
	for i, ef := range expectedFiles {
		af := actualFiles[i]
		efRelPath := ef.Path[len(expected):]
		afRelPath := af.Path[len(actual):]
		if efRelPath != afRelPath {
			t.Fatalf("File path '%s' does not match '%s'", afRelPath, efRelPath)
		}
		apply(t, ef.Path, af.Path, prefix)
	}
}

func assertCompatibleRepoFile(t *testing.T, expected string, actual string, prefix string) {
	if filepath.Base(expected) == filesName {
		// TODO: Check Filelist file
		// eFiles := loadFileList(expected)
		// aFiles := loadFileList(actual)
		// testutils.AssertLen(t, len(eFiles), aFiles, prefix)
		// for i, eFile := range eFiles {
		// 	eFile.Path = filepath.FromSlash(eFile.Path)
		// 	if eFile != aFiles[i] {
		// 		t.Fatal(prefix, "file entry do not match:", aFiles[i], ", expected:", eFile)
		// 	}
		// }
	} else if filepath.Base(expected) == recipeName {
		// TODO: Check Recipe files
		// eRecipe := loadRecipe(expected)
		// aRecipe := loadRecipe(actual)
		// testutils.AssertSame(t, eRecipe, aRecipe, prefix+"recipe")
	} else if filepath.Base(expected) == hashesName {
		// Hashes file is checked in TestHashes
	} else {
		// Chunk content file
		testutils.AssertSameFile(t, expected, actual, prefix)
	}
}

func assertChunkContent(t *testing.T, expected []byte, c Chunk, prefix string) {
	buf, err := io.ReadAll(c.Reader())
	if err != nil {
		t.Fatal(err)
	}
	testutils.AssertSame(t, expected, buf, prefix+" Chunk content")
}
