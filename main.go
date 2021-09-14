package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/n-peugnet/dna-backup/logger"
)

const (
	usage = "usage: dna-backup [<options>] [--] <source> <dest>\n\noptions:\n"
)

var (
	logLevel int
)

func init() {
	flag.IntVar(&logLevel, "v", 1, "log verbosity level (0-3)")
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage)
		flag.PrintDefaults()
	}
	flag.Parse()
	logger.Init(logLevel)
	if len(flag.Args()) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	source := os.Args[0]
	dest := os.Args[1]
	repo := NewRepo(dest)
	repo.Commit(source)
}
