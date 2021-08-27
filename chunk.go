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

type ChunkReader interface {
	io.Reader
	io.ByteReader
}

type Chunk struct {
	Repo  *Repo
	Id    *ChunkId
	Value []byte
}

func (c *Chunk) Read(buff []byte) (int, error) {
	r, err := c.Reader()
	if err != nil {
		return 0, err
	}
	return r.Read(buff)
}

func (c *Chunk) Reader() (ChunkReader, error) {
	if c.Value != nil {
		log.Printf("Chunk %d: Reading from in-memory value\n", c.Id)
		return bytes.NewReader(c.Value), nil
	}
	if c.Id != nil {
		log.Printf("Chunk %d: Reading from file\n", c.Id)
		return c.Id.Reader(c.Repo.path), nil
	}
	return nil, &ChunkError{"Uninitialized chunk"}
}

func (c *Chunk) isStored() bool {
	return c.Id != nil
}

type ChunkError struct {
	err string
}

func (e *ChunkError) Error() string {
	return fmt.Sprintf("Chunk error: %s", e.err)
}
