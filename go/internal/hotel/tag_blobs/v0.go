package tag_blobs

import "io"

type V0 struct{}

func (a *V0) Reset() {
}

func (a *V0) ResetWith(b V0) {
}

func (a *V0) GetFilterReader() (rc io.ReadCloser, err error) {
	return rc, err
}
