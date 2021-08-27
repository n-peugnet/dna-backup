package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"github.com/gabstv/go-bsdiff/pkg/bsdiff"
	"github.com/google/go-cmp/cmp"
)

func chunkCompare(t *testing.T, dataDir string, testFiles []string, chunkCount int) {
	reader, writer := io.Pipe()
	chunks := make(chan []byte)
	files := listFiles(dataDir)
	go concatFiles(files, writer)
	go chunkStream(reader, chunks)

	offset := 0
	buff := make([]byte, chunkSize*chunkCount)
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
		content := buff[i*chunkSize : (i+1)*chunkSize]
		if len(c) != chunkSize {
			t.Errorf("Chunk %d is not of chunkSize: %d", i, chunkSize)
		}
		if bytes.Compare(c, content) != 0 {
			t.Errorf("Chunk %d does not match file content", i)
			// for i, b := range c {
			// 	fmt.Printf("E: %d, A: %d\n", b, content[i])
			// }
			t.Log("Expected: ", c[:10], "...", c[chunkSize-10:])
			t.Log("Actual:", content)
		}
		i++
	}
	if i != chunkCount {
		t.Errorf("Incorrect number of chunks: %d, should be: %d", i, chunkCount)
	}
}

func TestReadFiles1(t *testing.T) {
	chunkCount := 590/chunkSize + 1
	dataDir := path.Join("test", "data", "logs", "1")
	files := []string{"logTest.log"}
	chunkCompare(t, dataDir, files, chunkCount)
}

func TestReadFiles2(t *testing.T) {
	chunkCount := 22899/chunkSize + 1
	dataDir := path.Join("test", "data", "logs", "2")
	files := []string{"csvParserTest.log", "slipdb.log"}
	chunkCompare(t, dataDir, files, chunkCount)
}

func TestReadFiles3(t *testing.T) {
	chunkCount := 119398/chunkSize + 1
	dataDir := path.Join("test", "data", "logs")
	files := []string{
		path.Join("1", "logTest.log"),
		path.Join("2", "csvParserTest.log"),
		path.Join("2", "slipdb.log"),
		path.Join("3", "indexingTreeTest.log"),
	}
	chunkCompare(t, dataDir, files, chunkCount)
}

func TestLoadChunks(t *testing.T) {
	resultDir := t.TempDir()
	dataDir := path.Join("test", "data", "logs")
	repo := NewRepo(resultDir)
	resultVersion := path.Join(resultDir, "00000")
	resultChunks := path.Join(resultVersion, chunksName)
	os.MkdirAll(resultChunks, 0775)
	reader1, writer1 := io.Pipe()
	reader2, writer2 := io.Pipe()
	chunks1 := make(chan []byte, 16)
	chunks2 := make(chan []byte, 16)
	chunks3 := make(chan Chunk, 16)
	files := listFiles(dataDir)
	go concatFiles(files, writer1)
	go concatFiles(files, writer2)
	go chunkStream(reader1, chunks1)
	go chunkStream(reader2, chunks2)
	storeChunks(resultChunks, chunks1)
	versions := []string{resultVersion}
	go repo.loadChunks(versions, chunks3)

	i := 0
	for c2 := range chunks2 {
		c3 := <-chunks3
		if bytes.Compare(c2, c3.Value) != 0 {
			t.Errorf("Chunk %d does not match file content", i)
			t.Log("Expected: ", c2[:10], "...")
			t.Log("Actual:", c3.Value)
		}
		i++
	}
}

func TestExtractNewChunks(t *testing.T) {
	chunks := []Chunk{
		{Value: []byte{'a'}},
		{Id: &ChunkId{0, 0}},
		{Value: []byte{'b'}},
		{Value: []byte{'c'}},
		{Id: &ChunkId{0, 1}},
	}
	newChunks := extractNewChunks(chunks)
	if len(newChunks) != 2 {
		t.Error("New chunks should contain 2 slices")
		t.Log("Actual: ", newChunks)
	}
	if len(newChunks[1]) != 2 {
		t.Error("New chunks second slice should contain 2 chunks")
		t.Log("Actual: ", newChunks[0])
	}
	if !cmp.Equal(newChunks[1][0], chunks[2]) {
		t.Error("New chunks do not match")
		t.Log("Expected: ", chunks[2])
		t.Log("Actual: ", newChunks[1][0])
	}
}

func TestStoreLoadFiles(t *testing.T) {
	resultDir := t.TempDir()
	dataDir := path.Join("test", "data", "logs")
	resultFiles := path.Join(resultDir, filesName)
	files1 := listFiles(dataDir)
	storeFileList(resultFiles, files1)
	files2 := loadFileList(resultFiles)
	if len(files1) != 4 {
		t.Errorf("Incorrect number of files: %d, should be %d\n", len(files1), 4)
	}
	for i, f := range files1 {
		if f != files2[i] {
			t.Errorf("Loaded file data %d does not match stored one", i)
			t.Log("Expected: ", f)
			t.Log("Actual: ", files2[i])
		}
	}
}

func TestBsdiff(t *testing.T) {
	resultDir := t.TempDir()
	dataDir := path.Join("test", "data", "logs")
	addedFile := path.Join(dataDir, "2", "slogTest.log")
	resultVersion := path.Join(resultDir, "00000")
	resultChunks := path.Join(resultVersion, chunksName)
	os.MkdirAll(resultChunks, 0775)
	reader, writer := io.Pipe()
	chunks := make(chan []byte, 16)
	files := listFiles(dataDir)
	go concatFiles(files, writer)
	go chunkStream(reader, chunks)
	storeChunks(resultChunks, chunks)

	input, _ := ioutil.ReadFile(path.Join(dataDir, "1", "logTest.log"))
	ioutil.WriteFile(addedFile, input, 0664)

	reader, writer = io.Pipe()
	oldChunks := make(chan Chunk, 16)
	files = listFiles(dataDir)
	repo := NewRepo(resultDir)
	versions := repo.loadVersions()
	go repo.loadChunks(versions, oldChunks)
	go concatFiles(files, writer)
	fingerprints, sketches := hashChunks(oldChunks)
	recipe := repo.matchStream(reader, fingerprints)
	buff := new(bytes.Buffer)
	r2, _ := recipe[2].Reader()
	r0, _ := recipe[0].Reader()
	bsdiff.Reader(r2, r0, buff)
	if len(buff.Bytes()) < 500 {
		t.Errorf("Bsdiff of chunk is too small: %d", len(buff.Bytes()))
	}
	if len(buff.Bytes()) >= chunkSize {
		t.Errorf("Bsdiff of chunk is too large: %d", len(buff.Bytes()))
	}
	newChunks := extractNewChunks(recipe)
	log.Println("Checking new chunks:", len(newChunks[0]))
	for _, c := range newChunks {
		findSimilarChunks(c, sketches)
	}

	os.Remove(addedFile)
}
