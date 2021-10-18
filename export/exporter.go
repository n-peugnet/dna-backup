package export

import "io"

type Version struct {
	Input
	Output
}

type Input struct {
	Chunks io.WriteCloser
	Recipe io.WriteCloser
	Files  io.WriteCloser
}

type Output struct {
	Chunks io.ReadCloser
	Recipe io.ReadCloser
	Files  io.ReadCloser
}

type Exporter interface {
	ExportVersion() (input Input, end <-chan bool)
}
