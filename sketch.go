package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"sync"

	"github.com/chmduquesne/rollinghash/rabinkarp64"
)

type Sketch []uint64

type ReadByteReader interface {
	io.Reader
	io.ByteReader
}

const fBytes = 8

// SketchChunk produces a sketch for a chunk based on wSize: the window size,
// sfCount: the number of super-features, and fCount: the number of feature
// per super-feature
func SketchChunk(chunk Chunk, chunkSize int, wSize int, sfCount int, fCount int) (Sketch, error) {
	var wg sync.WaitGroup
	var fSize = FeatureSize(chunkSize, sfCount, fCount)
	superfeatures := make([]uint64, 0, sfCount)
	features := make([]uint64, 0, fCount*sfCount)
	sfBuff := make([]byte, fBytes*fCount)
	r := chunk.Reader()
	for f := 0; f < chunk.Len()/fSize; f++ {
		var fBuff bytes.Buffer
		n, err := io.CopyN(&fBuff, r, int64(fSize))
		if err != nil {
			log.Println(n, err)
		}
		features = append(features, 0)
		wg.Add(1)
		go calcFeature(&wg, &fBuff, wSize, fSize, &features[f])
	}
	hasher := rabinkarp64.New()
	wg.Wait()
	for sf := 0; sf < len(features)/fCount; sf++ {
		for i := 0; i < fCount; i++ {
			binary.LittleEndian.PutUint64(sfBuff[i*fBytes:(i+1)*fBytes], features[i+sf*fCount])
		}
		hasher.Reset()
		hasher.Write(sfBuff)
		superfeatures = append(superfeatures, hasher.Sum64())
	}
	return superfeatures, nil
}

func calcFeature(wg *sync.WaitGroup, r ReadByteReader, wSize int, fSize int, result *uint64) {
	defer wg.Done()
	hasher := rabinkarp64.New()
	n, err := io.CopyN(hasher, r, int64(wSize))
	if err != nil {
		log.Println(n, err)
	}
	max := hasher.Sum64()
	for w := 0; w < fSize-wSize; w++ {
		b, _ := r.ReadByte()
		hasher.Roll(b)
		h := hasher.Sum64()
		if h > max {
			max = h
		}
	}
	*result = max
}

func SuperFeatureSize(chunkSize int, sfCount int, fCount int) int {
	return FeatureSize(chunkSize, sfCount, fCount) * sfCount
}

func FeatureSize(chunkSize int, sfCount int, fCount int) int {
	return chunkSize / (sfCount * fCount)
}
