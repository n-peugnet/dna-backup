package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
)

type ChunkReader interface {
	io.Reader
	io.ByteReader
}

type Chunk interface {
	Reader() ChunkReader
	Len() int
}

type StoredChunk interface {
	Chunk
	Id() *ChunkId
}

type ChunkId struct {
	Ver int
	Idx uint64
}

func (i *ChunkId) Path(repo string) string {
	return path.Join(repo, fmt.Sprintf(versionFmt, i.Ver), chunksName, fmt.Sprintf(chunkIdFmt, i.Idx))
}

func (i *ChunkId) Reader(repo *Repo) ChunkReader {
	path := i.Path(repo.path)
	f, err := os.Open(path)
	if err != nil {
		log.Println("Cannot open chunk: ", path)
	}
	return bufio.NewReaderSize(f, repo.chunkSize)
}

func NewLoadedChunk(id *ChunkId, value []byte) *LoadedChunk {
	return &LoadedChunk{id: id, value: value}
}

type LoadedChunk struct {
	id    *ChunkId
	value []byte
}

func (c *LoadedChunk) Id() *ChunkId {
	return c.id
}

func (c *LoadedChunk) Reader() ChunkReader {
	// log.Printf("Chunk %d: Reading from in-memory value\n", c.id)
	return bytes.NewReader(c.value)
}

func (c *LoadedChunk) Len() int {
	return len(c.value)
}

func NewChunkFile(repo *Repo, id *ChunkId) *ChunkFile {
	return &ChunkFile{repo: repo, id: id}
}

type ChunkFile struct {
	repo *Repo
	id   *ChunkId
}

func (c *ChunkFile) Id() *ChunkId {
	return c.id
}

func (c *ChunkFile) Reader() ChunkReader {
	// log.Printf("Chunk %d: Reading from file\n", c.id)
	return c.id.Reader(c.repo)
}

func (c *ChunkFile) Len() int {
	path := c.id.Path(c.repo.path)
	info, err := os.Stat(path)
	if err != nil {
		log.Println("Chunk: could not stat file:", path)
	}
	return int(info.Size())
}

func NewTempChunk(value []byte) *TempChunk {
	return &TempChunk{value: value}
}

type TempChunk struct {
	value []byte
}

func (c *TempChunk) Reader() ChunkReader {
	return bytes.NewReader(c.value)
}

func (c *TempChunk) Len() int {
	return len(c.value)
}

func (c *TempChunk) AppendFrom(r io.Reader) {
	buff, err := io.ReadAll(r)
	if err != nil {
		println("Chunk: error appending to temp chunk:", err)
	}
	c.value = append(c.value, buff...)
}
