package main

// Defined as var to prevent from using them as const as I want to keep
// beeing able to change tkem at runtime.
var (
	chunkSize  = 8 << 10
	chunksName = "chunks"
	chunkIdFmt = "%015d"
	versionFmt = "%05d"
	filesName  = "files"
)
