package ohio

import (
	"io"
	"unicode/utf8"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func TeeRuneReader(
	runeReader io.RuneReader,
	writer io.Writer,
) teeRuneReader {
	return teeRuneReader{
		runeReader: runeReader,
		writer:     writer,
	}
}

type teeRuneReader struct {
	runeReader io.RuneReader
	writer     io.Writer
}

func (tee teeRuneReader) ReadRune() (r rune, size int, err error) {
	r, size, err = tee.runeReader.ReadRune()

	b := make([]byte, utf8.UTFMax)
	n := utf8.EncodeRune(b, r)

	if n != size {
		err = errors.Join(
			err,
			errors.Errorf("read rune size does not match encoded size. expected %d, but got %d", size, n),
		)

		return r, size, err
	}

	if err != nil {
		err = errors.Wrap(err)
	}

	if _, errWrite := tee.writer.Write(b[:n]); errWrite != nil {
		err = errors.Join(err, errors.Wrap(errWrite))
		return r, size, err
	}

	return r, size, err
}
