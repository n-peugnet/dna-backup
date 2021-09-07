package main

import (
	"bytes"
	"errors"
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

type StorerChunk interface {
	Chunk
	Store(path string) error
}

type ChunkId struct {
	Ver int
	Idx uint64
}

func (i *ChunkId) Path(repo string) string {
	return path.Join(repo, fmt.Sprintf(versionFmt, i.Ver), chunksName, fmt.Sprintf(chunkIdFmt, i.Idx))
}

func (i *ChunkId) Reader(repo *Repo) io.ReadSeeker {
	path := i.Path(repo.path)
	f, err := os.Open(path)
	if err != nil {
		log.Println("Cannot open chunk: ", path)
	}
	return f
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

func (c *LoadedChunk) Store(path string) error {
	return storeChunk(c.Reader(), c.Id.Path(path))
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
	return c.Id.Reader(c.repo)
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
	c.repo.Patcher().Patch(c.Source.Reader(c.repo), &buff, bytes.NewReader(c.Patch))
	return bytes.NewReader(buff.Bytes())
}

// TODO: Maybe return the size of the patch instead ?
func (c *DeltaChunk) Len() int {
	return c.Size
}

func storeChunk(r io.Reader, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return errors.New(fmt.Sprintf("Error creating chunk for '%s'; %s\n", path, err))
	}
	n, err := io.Copy(file, r)
	if err != nil {
		return errors.New(fmt.Sprintf("Error writing chunk content for '%s', written %d bytes: %s\n", path, n, err))
	}
	if err := file.Close(); err != nil {
		return errors.New(fmt.Sprintf("Error closing chunk for '%s': %s\n", path, err))
	}
	return nil
}
