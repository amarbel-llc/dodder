package ohio_buffer

import (
	"bytes"
)

func Copy(dst, src *bytes.Buffer) (err error) {
	dst.Reset()
	dst.Grow(src.Len())
	n, err := dst.Write(src.Bytes())
	return MakeErrLength(int64(src.Len()), int64(n), err)
}
