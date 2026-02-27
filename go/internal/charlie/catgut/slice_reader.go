package catgut

import "io"

type SliceReader struct {
	slice  Slice
	offset int
}

func MakeSliceReader(sl Slice) *SliceReader {
	return &SliceReader{
		slice: sl,
	}
}

func (sr *SliceReader) ResetWith(sl Slice) {
	sr.slice = sl
	sr.offset = 0
}

func (sr *SliceReader) Read(p []byte) (n int, err error) {
	var n1 int

	switch {
	case sr.offset < sr.slice.LenFirst():
		n1 = copy(p, sr.slice.First()[sr.offset:])
		n += n1
		sr.offset += n1

		if n == len(p) || len(p) <= sr.slice.LenFirst() {
			return n, err
		}

		fallthrough

	case sr.offset < sr.slice.Len():
		n1 = copy(p, sr.slice.Second()[sr.offset-sr.slice.LenFirst():])
		n += n1
		sr.offset += n1
	}

	if sr.offset == sr.slice.Len() {
		err = io.EOF
		return n, err
	}

	return n, err
}
