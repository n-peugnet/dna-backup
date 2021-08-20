package main

import (
	"os"
)

func main() {
	path := "."

	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	files := make(chan File)
	chunks := make(chan []byte)
	go ListFiles(path, files)
	go ReadFiles(files, chunks)
	StoreChunks(".", chunks)
}
