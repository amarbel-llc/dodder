package ohio

import (
	"bytes"
	"io"
	"unicode/utf8"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

// TODO add tests
func DebugRuneScanner(
	runeReader io.RuneScanner,
) *teeRuneScanner {
	return &teeRuneScanner{
		runeScanner: runeReader,
	}
}

type teeRuneScanner struct {
	lastWidth   int
	runeScanner io.RuneScanner
	buffer      bytes.Buffer
}

func (tee *teeRuneScanner) UnreadRune() (err error) {
	if err = tee.runeScanner.UnreadRune(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	tee.buffer.Truncate(tee.buffer.Len() - tee.lastWidth)
	tee.lastWidth = utf8.RuneError

	return err
}

func (tee *teeRuneScanner) ReadRune() (r rune, size int, err error) {
	if r, size, err = tee.runeScanner.ReadRune(); err != nil {
		if err == io.EOF {
			return r, size, err
		} else {
			err = errors.Wrap(err)
			return r, size, err
		}
	}

	tee.lastWidth = size

	if _, err = tee.buffer.WriteRune(r); err != nil {
		err = errors.Wrap(err)
		return r, size, err
	}

	return r, size, err
}

func (tee *teeRuneScanner) GetBytes() []byte {
	return tee.buffer.Bytes()
}
