package ohio

import (
	"bytes"
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/alfa/pool"
)

func WriteKeySpaceValueNewlineString(
	writer io.Writer,
	key, value string,
) (n int, err error) {
	return WriteStrings(writer, key, " ", value, "\n")
}

func WriteKeySpaceValueNewline(
	writer io.Writer,
	key string, value []byte,
) (n int64, err error) {
	var (
		n1         int64
		byteReader *bytes.Reader
	)

	stringReader, repool1 := pool.GetStringReader(key)
	defer repool1()
	n1, err = stringReader.WriteTo(writer)
	n += n1

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	stringReader2, repool2 := pool.GetStringReader(" ")
	defer repool2()

	n1, err = stringReader2.WriteTo(writer)
	n += n1

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	byteReader = bytes.NewReader(value)

	n1, err = byteReader.WriteTo(writer)
	n += n1

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	stringReader3, repool3 := pool.GetStringReader("\n")
	defer repool3()

	n1, err = stringReader3.WriteTo(writer)
	n += n1

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func WriteStrings(
	w io.Writer,
	ss ...string,
) (n int, err error) {
	for _, s := range ss {
		var n1 int

		n1, err = io.WriteString(w, s)
		n += n1

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	return n, err
}
