package main

import (
	"encoding/binary"
	"io"
	"log"

	"github.com/chmduquesne/rollinghash/rabinkarp64"
)

type Sketch []uint64

const fBytes = 8

// SketchChunk produces a sketch for a chunk based on wSize: the window size,
// sfCount: the number of super-features, and fCount: the number of feature
// per super-feature
func SketchChunk(chunk Chunk, wSize int, sfCount int, fCount int) (Sketch, error) {
	var fSize = chunkSize / (sfCount * fCount)
	superfeatures := make([]uint64, 0, sfCount)
	features := make([]uint64, 0, fCount*sfCount)
	buff := make([]byte, fBytes*fCount)
	r := chunk.Reader()
	hasher := rabinkarp64.New()
	for f := 0; f < chunk.Len()/fSize; f++ {
		hasher.Reset()
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
		features = append(features, max)
	}
	for sf := 0; sf < len(features)/fCount; sf++ {
		for i := 0; i < fCount; i++ {
			binary.LittleEndian.PutUint64(buff[i*fBytes:(i+1)*fBytes], features[i+sf*fCount])
		}
		hasher.Reset()
		hasher.Write(buff)
		superfeatures = append(superfeatures, hasher.Sum64())
	}
	return superfeatures, nil
}
