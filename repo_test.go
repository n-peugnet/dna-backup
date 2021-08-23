package main

import (
	"bytes"
	"log"
	"os"
	"path"
	"testing"
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
	files := ListFiles(dataDir)
	go ReadFiles(files, chunks)

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
	resultChunks := path.Join(resultVersion, "chunks")
	os.MkdirAll(resultChunks, 0775)
	chunks1 := make(chan []byte, 16)
	chunks2 := make(chan []byte, 16)
	chunks3 := make(chan Chunk, 16)
	files := ListFiles(dataDir)
	go ReadFiles(files, chunks1)
	go ReadFiles(files, chunks2)
	StoreChunks(resultChunks, chunks1)
	versions := []string{resultVersion}
	go LoadChunks(versions, chunks3)

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
	resultFiles := path.Join(resultDir, "files")
	files1 := ListFiles(dataDir)
	StoreFiles(resultFiles, files1)
	files2 := LoadFiles(resultFiles)
	for i, f := range files1 {
		if f != files2[i] {
			t.Errorf("Loaded file data %d does not match stored one", i)
			t.Log("Expected: ", f)
			t.Log("Result: ", files2[i])
		}
	}
}
