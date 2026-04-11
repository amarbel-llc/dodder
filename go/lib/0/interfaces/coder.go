package interfaces

import (
	"bufio"
	"io"
)

type (
	DecoderFrom[OBJECT any, READER any] interface {
		DecodeFrom(OBJECT, READER) (int64, error)
	}

	EncoderTo[OBJECT any, READER any] interface {
		EncodeTo(OBJECT, READER) (int64, error)
	}

	Coder[OBJECT any, READER any, WRITER any] interface {
		DecoderFrom[OBJECT, READER]
		EncoderTo[OBJECT, WRITER]
	}

	DecoderFromBufferedReader[OBJECT any] = DecoderFrom[OBJECT, *bufio.Reader]
	EncoderToBufferedWriter[OBJECT any]   = EncoderTo[OBJECT, *bufio.Writer]
	CoderBufferedReadWriter[OBJECT any]   = Coder[OBJECT, *bufio.Reader, *bufio.Writer]

	DecoderFromReader[OBJECT any] = DecoderFrom[OBJECT, io.Reader]
	EncoderToWriter[OBJECT any]   = EncoderTo[OBJECT, io.Writer]
	CoderReadWriter[OBJECT any]   = Coder[OBJECT, io.Reader, io.Writer]

	StringEncoderTo[T any] interface {
		EncodeStringTo(T, WriterAndStringWriter) (int64, error)
	}

	StringCoder[T any] interface {
		DecoderFromReader[T]
		StringEncoderTo[T]
	}
)
