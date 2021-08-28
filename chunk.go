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
}

type StoredChunk interface {
	Chunk
	Id() *ChunkId
}

type ChunkId struct {
	Ver int
	Idx uint64
}

func (i *ChunkId) Reader(repo string) ChunkReader {
	p := path.Join(repo, fmt.Sprintf(versionFmt, i.Ver), chunksName, fmt.Sprintf(chunkIdFmt, i.Idx))
	f, err := os.Open(p)
	if err != nil {
		log.Printf("Cannot open chunk %s\n", p)
	}
	return bufio.NewReaderSize(f, chunkSize)
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
	return c.id.Reader(c.repo.path)
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
