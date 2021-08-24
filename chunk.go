package main

import (
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

func (i *ChunkId) Reader(repo string) io.Reader {
	p := path.Join(repo, fmt.Sprintf(versionFmt, i.Ver), chunksName, fmt.Sprintf(chunkIdFmt, i.Idx))
	f, err := os.Open(p)
	if err != nil {
		log.Printf("Cannot open chunk %s\n", p)
	}
	return f
}

type Chunk struct {
	Id    *ChunkId
	Value []byte
}

func (c *Chunk) Reader(repo string) (io.Reader, error) {
	if c.Value != nil {
		return bytes.NewReader(c.Value), nil
	}
	if c.Id != nil {
		return c.Id.Reader(repo), nil
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
