package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
)

type Chunk interface {
	Reader() io.ReadSeeker
	Len() int
}

type IdentifiedChunk interface {
	Chunk
	GetId() *ChunkId
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

func NewLoadedChunk(id *ChunkId, value []byte) *LoadedChunk {
	return &LoadedChunk{Id: id, value: value}
}

type LoadedChunk struct {
	Id    *ChunkId
	value []byte
}

func (c *LoadedChunk) GetId() *ChunkId {
	return c.Id
}

func (c *LoadedChunk) Reader() io.ReadSeeker {
	// log.Printf("Chunk %d: Reading from in-memory value\n", c.id)
	return bytes.NewReader(c.value)
}

func (c *LoadedChunk) Len() int {
	return len(c.value)
}

func (c *LoadedChunk) Bytes() []byte {
	return c.value
}

func NewStoredChunk(repo *Repo, id *ChunkId) *StoredChunk {
	return &StoredChunk{repo: repo, Id: id}
}

type StoredChunk struct {
	repo *Repo
	Id   *ChunkId
}

func (c *StoredChunk) GetId() *ChunkId {
	return c.Id
}

func (c *StoredChunk) Reader() io.ReadSeeker {
	// log.Printf("Chunk %d: Reading from file\n", c.id)
	return c.repo.LoadChunkContent(c.Id)
}

func (c *StoredChunk) Len() int {
	path := c.Id.Path(c.repo.path)
	info, err := os.Stat(path)
	if err != nil {
		log.Println("Chunk: could not stat file:", path)
	}
	return int(info.Size())
}

func NewTempChunk(value []byte) *TempChunk {
	return &TempChunk{Value: value}
}

type TempChunk struct {
	Value []byte
}

func (c *TempChunk) Reader() io.ReadSeeker {
	return bytes.NewReader(c.Value)
}

func (c *TempChunk) Len() int {
	return len(c.Value)
}

func (c *TempChunk) Bytes() []byte {
	return c.Value
}

func (c *TempChunk) AppendFrom(r io.Reader) {
	buff, err := io.ReadAll(r)
	if err != nil {
		println("Chunk: error appending to temp chunk:", err)
	}
	c.Value = append(c.Value, buff...)
}

type DeltaChunk struct {
	repo   *Repo
	Source *ChunkId
	Patch  []byte
	Size   int
}

func (c *DeltaChunk) Reader() io.ReadSeeker {
	var buff bytes.Buffer
	c.repo.Patcher().Patch(c.repo.LoadChunkContent(c.Source), &buff, bytes.NewReader(c.Patch))
	return bytes.NewReader(buff.Bytes())
}

// TODO: Maybe return the size of the patch instead ?
func (c *DeltaChunk) Len() int {
	return c.Size
}
