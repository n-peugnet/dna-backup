package main

import (
	"archive/tar"
	"io"
	"os"

	"github.com/n-peugnet/dna-backup/logger"
)

func streamFilesTar(files []File, stream io.WriteCloser) {
	tarStream := tar.NewWriter(stream)
	for _, f := range files {
		file, err := os.Open(f.Path)
		if err != nil {
			logger.Error(err)
			continue
		}
		stat, err := file.Stat()
		if err != nil {
			logger.Errorf("getting stat of file '%s': %s", f.Path, err)
			continue
		}
		hdr, err := tar.FileInfoHeader(stat, "")
		if err != nil {
			logger.Errorf("creating tar header for file '%s': %s", f.Path, err)
			continue
		}
		if err := tarStream.WriteHeader(hdr); err != nil {
			logger.Panicf("writing tar header to stream for file '%s': %s", f.Path, err)
		}
		if _, err := io.Copy(tarStream, file); err != nil {
			logger.Panicf("writing file to stream '%s': %s", f.Path, err)
		}
	}
	if err := tarStream.Close(); err != nil {
		logger.Panic("closing tar stream:", err)
	}
	if err := stream.Close(); err != nil {
		logger.Panic("closing stream:", err)
	}
}
