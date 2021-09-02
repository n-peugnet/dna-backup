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

type IdentifiedChunk interface {
	Chunk
	Id() *ChunkId
}

type BufferedChunk interface {
	Chunk
	Bytes() []byte
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

func (c *LoadedChunk) Bytes() []byte {
	return c.value
}

func NewStoredFile(repo *Repo, id *ChunkId) *StoredChunk {
	return &StoredChunk{repo: repo, id: id}
}

type StoredChunk struct {
	repo *Repo
	id   *ChunkId
}

func (c *StoredChunk) Id() *ChunkId {
	return c.id
}

func (c *StoredChunk) Reader() ChunkReader {
	// log.Printf("Chunk %d: Reading from file\n", c.id)
	return c.id.Reader(c.repo)
}

func (c *StoredChunk) Len() int {
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

func (c *TempChunk) Bytes() []byte {
	return c.value
}

func (c *TempChunk) AppendFrom(r io.Reader) {
	buff, err := io.ReadAll(r)
	if err != nil {
		println("Chunk: error appending to temp chunk:", err)
	}
	c.value = append(c.value, buff...)
}

type DeltaChunk struct {
	repo   *Repo
	source *ChunkId
	patch  []byte
	size   int
}

func (c *DeltaChunk) Reader() ChunkReader {
	var buff bytes.Buffer
	c.repo.Patcher().Patch(c.source.Reader(c.repo), &buff, bytes.NewReader(c.patch))
	return &buff
}

// TODO: Maybe return the size of the patch instead ?
func (c *DeltaChunk) Len() int {
	return c.size
}
