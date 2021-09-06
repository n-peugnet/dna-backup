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
	Id() *ChunkId
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
	return &LoadedChunk{id: id, value: value}
}

type LoadedChunk struct {
	id    *ChunkId
	value []byte
}

func (c *LoadedChunk) Id() *ChunkId {
	return c.id
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
	return storeChunk(c.Reader(), c.id.Path(path))
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

func (c *StoredChunk) Reader() io.ReadSeeker {
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

func (c *TempChunk) Reader() io.ReadSeeker {
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

func (c *DeltaChunk) Reader() io.ReadSeeker {
	var buff bytes.Buffer
	c.repo.Patcher().Patch(c.source.Reader(c.repo), &buff, bytes.NewReader(c.patch))
	return bytes.NewReader(buff.Bytes())
}

// TODO: Maybe return the size of the patch instead ?
func (c *DeltaChunk) Len() int {
	return c.size
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
