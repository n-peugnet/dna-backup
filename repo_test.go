package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"testing"
)

func chunkCompare(t *testing.T, dataDir string, repo *Repo, testFiles []string, chunkCount int) {
	reader, writer := io.Pipe()
	chunks := make(chan []byte)
	files := listFiles(dataDir)
	go concatFiles(files, writer)
	go repo.chunkStream(reader, chunks)

	offset := 0
	buff := make([]byte, repo.chunkSize*chunkCount)
	for _, f := range testFiles {
		content, err := os.ReadFile(path.Join(dataDir, f))
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

func TestReadFiles1(t *testing.T) {
	repo := NewRepo("")
	chunkCount := 590/repo.chunkSize + 1
	dataDir := path.Join("testdata", "logs", "1")
	files := []string{"logTest.log"}
	chunkCompare(t, dataDir, repo, files, chunkCount)
}

func TestReadFiles2(t *testing.T) {
	repo := NewRepo("")
	chunkCount := 22899/repo.chunkSize + 1
	dataDir := path.Join("testdata", "logs", "2")
	files := []string{"csvParserTest.log", "slipdb.log"}
	chunkCompare(t, dataDir, repo, files, chunkCount)
}

func TestReadFiles3(t *testing.T) {
	repo := NewRepo("")
	chunkCount := 119398/repo.chunkSize + 1
	dataDir := path.Join("testdata", "logs")
	files := []string{
		path.Join("1", "logTest.log"),
		path.Join("2", "csvParserTest.log"),
		path.Join("2", "slipdb.log"),
		path.Join("3", "indexingTreeTest.log"),
	}
	chunkCompare(t, dataDir, repo, files, chunkCount)
}

func TestLoadChunks(t *testing.T) {
	resultDir := t.TempDir()
	dataDir := path.Join("testdata", "logs")
	repo := NewRepo(resultDir)
	resultVersion := path.Join(resultDir, "00000")
	resultChunks := path.Join(resultVersion, chunksName)
	os.MkdirAll(resultChunks, 0775)
	reader1, writer1 := io.Pipe()
	reader2, writer2 := io.Pipe()
	chunks1 := make(chan []byte, 16)
	chunks2 := make(chan []byte, 16)
	chunks3 := make(chan IdentifiedChunk, 16)
	files := listFiles(dataDir)
	go concatFiles(files, writer1)
	go concatFiles(files, writer2)
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

func TestStoreLoadFiles(t *testing.T) {
	resultDir := t.TempDir()
	dataDir := path.Join("testdata", "logs")
	resultFiles := path.Join(resultDir, filesName)
	files1 := listFiles(dataDir)
	storeFileList(resultFiles, files1)
	files2 := loadFileList(resultFiles)
	assertLen(t, 4, files1, "Files:")
	for i, f := range files1 {
		if f != files2[i] {
			t.Errorf("Loaded file data %d does not match stored one", i)
			t.Log("Expected: ", f)
			t.Log("Actual: ", files2[i])
		}
	}
}

func prepareChunks(dataDir string, repo *Repo, streamFunc func([]File, io.WriteCloser)) {
	resultVersion := path.Join(repo.path, "00000")
	resultChunks := path.Join(resultVersion, chunksName)
	os.MkdirAll(resultChunks, 0775)
	reader := getDataStream(dataDir, streamFunc)
	chunks := make(chan []byte, 16)
	go repo.chunkStream(reader, chunks)
	storeChunks(resultChunks, chunks)
}

func getDataStream(dataDir string, streamFunc func([]File, io.WriteCloser)) io.Reader {
	reader, writer := io.Pipe()
	files := listFiles(dataDir)
	go streamFunc(files, writer)
	return reader
}

func TestBsdiff(t *testing.T) {
	resultDir := t.TempDir()
	repo := NewRepo(resultDir)
	dataDir := path.Join("testdata", "logs")
	addedFile1 := path.Join(dataDir, "2", "slogTest.log")
	addedFile2 := path.Join(dataDir, "3", "slogTest.log")
	// Store initial chunks
	prepareChunks(dataDir, repo, concatFiles)

	// Modify data
	ioutil.WriteFile(addedFile1, []byte("hello"), 0664)
	defer os.Remove(addedFile1)
	ioutil.WriteFile(addedFile2, make([]byte, 4000), 0664)
	defer os.Remove(addedFile2)

	// Load previously stored chunks
	oldChunks := make(chan IdentifiedChunk, 16)
	versions := repo.loadVersions()
	newVersion := len(versions)
	go repo.loadChunks(versions, oldChunks)
	repo.hashChunks(oldChunks)

	// Read new data
	reader := getDataStream(dataDir, concatFiles)
	recipe := repo.matchStream(reader, newVersion)
	newChunks := extractDeltaChunks(recipe)
	assertLen(t, 2, newChunks, "New delta chunks:")
	for _, c := range newChunks {
		log.Println("Patch size:", len(c.Patch))
		if len(c.Patch) >= repo.chunkSize/10 {
			t.Errorf("Bsdiff of chunk is too large: %d", len(c.Patch))
		}
	}
}

func TestCommit(t *testing.T) {
	dest := t.TempDir()
	source := path.Join("testdata", "logs")
	repo := NewRepo(dest)
	repo.Commit(source)
	recipe := loadRecipe(path.Join(dest, "00000", recipeName))
	log.Println(recipe)
}

func TestRestore(t *testing.T) {
	dest := t.TempDir()
	source := path.Join("testdata", "repo_8k")
	repo := NewRepo(source)
	repo.Restore(dest)
	destFiles := listFiles(dest)
	sourceFiles := listFiles(path.Join("testdata", "logs"))
	sfCount := len(sourceFiles)
	if sfCount <= 0 {
		t.Fatalf("No source files: %d", sfCount)
	}
	dfCount := len(destFiles)
	if sfCount != dfCount {
		t.Fatalf("Incorrect number for destination files: %d, should be %d", dfCount, sfCount)
	}
	for i, sf := range sourceFiles {
		sfContent, err := os.ReadFile(sf.Path)
		if err != nil {
			t.Fatalf("Error reading from source file '%s': %s", sf.Path, err)
		}
		df := destFiles[i]
		dfContent, err := os.ReadFile(df.Path)
		if err != nil {
			t.Fatalf("Error reading from source file '%s': %s", df.Path, err)
		}
		if bytes.Compare(sfContent, dfContent) != 0 {
			t.Errorf("File content of '%s' does not match '%s'", df.Path, sf.Path)
		}
	}
}

func assertLen(t *testing.T, expected int, actual interface{}, prefix string) {
	s := reflect.ValueOf(actual)
	if s.Len() != expected {
		t.Fatal(prefix, "incorrect length, expected:", expected, ", actual:", s.Len())
	}
}

func assertSameSlice(t *testing.T, expected []byte, actual []byte, prefix string) {
	assertLen(t, len(expected), actual, prefix)
	for i := 0; i < len(expected); i++ {
		if expected[i] != actual[i] {
			t.Fatal(prefix, "incorrect value", i, ", expected:", expected[i], ", actual:", actual[i])
		}
	}
}

func assertChunkContent(t *testing.T, expected []byte, c Chunk, prefix string) {
	buf, err := io.ReadAll(c.Reader())
	if err != nil {
		t.Fatal(err)
	}
	assertSameSlice(t, expected, buf, prefix+" Chunk content")
}
