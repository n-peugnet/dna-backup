package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"github.com/gabstv/go-bsdiff/pkg/bsdiff"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	log.SetFlags(log.Lshortfile)
}

func shutdown() {}

func prepareResult() string {
	result := path.Join("test", "result")
	os.RemoveAll(result)
	os.MkdirAll(result, 0775)
	return result
}

func chunkCompare(t *testing.T, dataDir string, testFiles []string, chunkCount int) {

	chunks := make(chan []byte)
	files := listFiles(dataDir)
	go readFiles(files, chunks)

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
			t.Log("Expected: ", c[:10], "...")
			t.Log("Result:", content)
		}
		i++
	}
	if i != chunkCount {
		t.Errorf("Incorrect number of chunks: %d, should be: %d", i, chunkCount)
	}
}

func TestReadFiles1(t *testing.T) {
	chunkCount := 1
	dataDir := path.Join("test", "data", "logs.1")
	files := []string{"logTest.log"}
	chunkCompare(t, dataDir, files, chunkCount)
}

func TestReadFiles2(t *testing.T) {
	chunkCount := 3
	dataDir := path.Join("test", "data", "logs.2")
	files := []string{"csvParserTest.log", "slipdb.log"}
	chunkCompare(t, dataDir, files, chunkCount)
}

func TestReadFiles3(t *testing.T) {
	chunkCount := 15
	dataDir := path.Join("test", "data")
	files := []string{
		path.Join("logs.1", "logTest.log"),
		path.Join("logs.2", "csvParserTest.log"),
		path.Join("logs.2", "slipdb.log"),
		path.Join("logs.3", "indexingTreeTest.log"),
	}
	chunkCompare(t, dataDir, files, chunkCount)
}

func TestLoadChunks(t *testing.T) {
	resultDir := prepareResult()
	dataDir := path.Join("test", "data")
	resultVersion := path.Join(resultDir, "00000")
	resultChunks := path.Join(resultVersion, chunksName)
	os.MkdirAll(resultChunks, 0775)
	chunks1 := make(chan []byte, 16)
	chunks2 := make(chan []byte, 16)
	chunks3 := make(chan Chunk, 16)
	files := listFiles(dataDir)
	go readFiles(files, chunks1)
	go readFiles(files, chunks2)
	storeChunks(resultChunks, chunks1)
	versions := []string{resultVersion}
	go loadChunks(versions, chunks3)

	i := 0
	for c2 := range chunks2 {
		c3 := <-chunks3
		if bytes.Compare(c2, c3.Value) != 0 {
			t.Errorf("Chunk %d does not match file content", i)
			t.Log("Expected: ", c2[:10], "...")
			t.Log("Result:", c3.Value)
		}
		i++
	}
}

func TestStoreLoadFiles(t *testing.T) {
	resultDir := prepareResult()
	dataDir := path.Join("test", "data")
	resultFiles := path.Join(resultDir, filesName)
	files1 := listFiles(dataDir)
	storeFiles(resultFiles, files1)
	files2 := loadFiles(resultFiles)
	for i, f := range files1 {
		if f != files2[i] {
			t.Errorf("Loaded file data %d does not match stored one", i)
			t.Log("Expected: ", f)
			t.Log("Result: ", files2[i])
		}
	}
}

func TestBsdiff(t *testing.T) {
	resultDir := prepareResult()
	dataDir := path.Join("test", "data")
	addedFile := path.Join(dataDir, "logs.2", "slogTest.log")
	resultVersion := path.Join(resultDir, "00000")
	resultChunks := path.Join(resultVersion, chunksName)
	os.MkdirAll(resultChunks, 0775)
	chunks := make(chan []byte, 16)
	files := listFiles(dataDir)
	go readFiles(files, chunks)
	storeChunks(resultChunks, chunks)

	input, _ := ioutil.ReadFile(path.Join(dataDir, "logs.1", "logTest.log"))
	ioutil.WriteFile(addedFile, input, 0664)

	newChunks := make(chan []byte, 16)
	oldChunks := make(chan Chunk, 16)
	files = listFiles(dataDir)
	repo := NewRepo(resultDir)
	versions := repo.loadVersions()
	go loadChunks(versions, oldChunks)
	go readFiles(files, newChunks)
	hashes := hashChunks(oldChunks)
	recipe := repo.matchChunks(newChunks, hashes)
	buff := new(bytes.Buffer)
	r2, _ := recipe[2].Reader(repo.path)
	r0, _ := recipe[0].Reader(repo.path)
	bsdiff.Reader(r2, r0, buff)
	if len(buff.Bytes()) < 500 {
		t.Errorf("Bsdiff of chunk is too small: %d", len(buff.Bytes()))
	}
	if len(buff.Bytes()) >= chunkSize {
		t.Errorf("Bsdiff of chunk is too large: %d", len(buff.Bytes()))
	}

	os.Remove(addedFile)
}
