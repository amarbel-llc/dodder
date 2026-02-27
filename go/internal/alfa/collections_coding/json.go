package collections_coding

import (
	"encoding/json"
	"io"
)

type EncoderJson[T any] struct {
	enc *json.Encoder
}

func MakeEncoderJson[T any](out io.Writer) EncoderJson[T] {
	return EncoderJson[T]{
		enc: json.NewEncoder(out),
	}
}

func (e EncoderJson[T]) Encode(o *T) (n int64, err error) {
	err = e.enc.Encode(o)
	return n, err
}
