package main

import (
	"log"
	"os"
	"testing"

	"github.com/n-peugnet/dna-backup/logger"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	logger.SetFlags(log.Lshortfile)
}

func shutdown() {}
