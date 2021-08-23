package main

import (
	"fmt"
	"os"
)

func main() {

	if len(os.Args) != 3 {
		fmt.Println("usage: dna-backup <source> <dest>")
		os.Exit(1)
	}

	source := os.Args[1]
	dest := os.Args[2]
	repo := NewRepo(dest)
	repo.Commit(source)
}
