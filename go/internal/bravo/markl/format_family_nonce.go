package markl

import (
	"io"

	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

func NonceGenerate(rand io.Reader, size int) (bites []byte, err error) {
	bites = make([]byte, size)

	if _, err = rand.Read(bites); err != nil {
		err = errors.Wrap(err)
		return bites, err
	}

	return bites, err
}

func NonceGenerate32(rand io.Reader) (bites []byte, err error) {
	return NonceGenerate(rand, 32)
}
