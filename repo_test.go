package main

import (
	"bytes"
	"os"
	"path"
	"testing"
)

func prepareResult() {
	result := path.Join("test", "result")
	os.RemoveAll(result)
	os.MkdirAll(result, 0775)
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
			t.Log(c)
			t.Log(content)
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
	prepareResult()
	dataDir := path.Join("test", "data")
	resultDir := path.Join("test", "result")
	chunks1 := make(chan []byte, 16)
	chunks2 := make(chan []byte, 16)
	chunks3 := make(chan []byte, 16)
	files := ListFiles(dataDir)
	go ReadFiles(files, chunks1)
	go ReadFiles(files, chunks2)
	StoreChunks(resultDir, chunks1)
	go LoadChunks(resultDir, chunks3)

	i := 0
	for c2 := range chunks2 {
		c3 := <-chunks3
		if bytes.Compare(c2, c3) != 0 {
			t.Errorf("Chunk %d does not match file content", i)
			t.Log(c2)
			t.Log(c3)
		}
		i++
	}
}
