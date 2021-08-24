package main

import (
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	log.SetFlags(log.Lshortfile)
}

func shutdown() {}
