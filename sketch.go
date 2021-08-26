package main

import (
	"encoding/binary"
	"io"

	"github.com/chmduquesne/rollinghash/rabinkarp64"
)

// SketchChunk produces a sketch for a chunk based on wSize: the window size,
// sfCount: the number of super-features, and fCount: the number of feature
// per super-feature
func SketchChunk(chunk Chunk, wSize int, sfCount int, fCount int) ([]uint64, error) {
	var fSize = chunkSize / (sfCount * fCount)
	superfeatures := make([]uint64, 0, sfCount)
	features := make([]uint64, 0, fCount)
	buff := make([]byte, 8*fCount)
	r, err := chunk.Reader()
	if err != nil {
		return nil, err
	}
	hasher := rabinkarp64.New()
	for sf := 0; sf < sfCount; sf++ {
		features = features[:0]
		for f := 0; f < fCount; f++ {
			hasher.Reset()
			io.CopyN(hasher, r, int64(wSize))
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
		for i, f := range features {
			binary.LittleEndian.PutUint64(buff[i*8:i*8+8], f)
		}
		hasher.Reset()
		hasher.Write(buff)
		superfeatures = append(superfeatures, hasher.Sum64())
	}
	return superfeatures, nil
}
