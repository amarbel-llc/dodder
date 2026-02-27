package triple_hyphen_io

import "io"

type MetadataWriterTo interface {
	io.WriterTo
	HasMetadataContent() bool
}

type readerState int

const (
	readerStateEmpty = readerState(iota)
	readerStateFirstBoundary
	readerStateSecondBoundary
)
