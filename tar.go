package main

import (
	"archive/tar"
	"io"
	"log"
	"os"
)

func streamFilesTar(files []File, stream io.WriteCloser) {
	tarStream := tar.NewWriter(stream)
	for _, f := range files {
		file, err := os.Open(f.Path)
		if err != nil {
			log.Printf("Error reading file '%s': %s\n", f.Path, err)
			continue
		}
		stat, err := file.Stat()
		if err != nil {
			log.Printf("Error getting stat of file '%s': %s\n", f.Path, err)
			continue
		}
		hdr, err := tar.FileInfoHeader(stat, "")
		if err != nil {
			log.Printf("Error creating tar header for file '%s': %s\n", f.Path, err)
			continue
		}
		if err := tarStream.WriteHeader(hdr); err != nil {
			log.Printf("Error writing tar header to stream for file '%s': %s\n", f.Path, err)
			continue
		}
		if _, err := io.Copy(tarStream, file); err != nil {
			log.Printf("Error writing file to stream '%s': %s\n", f.Path, err)
			continue
		}
	}
	if err := tarStream.Close(); err != nil {
		log.Fatal("Error closing tar stream:", err)
	}
	if err := stream.Close(); err != nil {
		log.Fatal("Error closing stream:", err)
	}
}
