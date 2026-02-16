package delim_reader

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
)

var delimReaderPool interfaces.PoolPtr[delimReader, *delimReader]

func init() {
	delimReaderPool = pool.MakeWithResetable[delimReader]()
}

// Not safe for parallel use
type DelimReader interface {
	io.Reader
	N() int64
	Segments() int64
	IsEOF() bool
	ResetWith(dr delimReader)
	Reset()
	ReadOneString() (str string, err error)
	ReadOneKeyValue(sep string) (key, val string, err error)
}

type delimReader struct {
	delim byte
	bufio.Reader
	n         int64
	lastReadN int
	segments  int64
	eof       bool
}

func MakeDelimReader(
	delim byte,
	ioReader io.Reader,
) (reader *delimReader, repool interfaces.FuncRepool) {
	reader, repool = delimReaderPool.GetWithRepool()
	reader.Reader.Reset(ioReader)
	reader.delim = delim

	return reader, repool
}

func (reader *delimReader) N() int64 {
	return reader.n
}

func (reader *delimReader) Segments() int64 {
	return reader.segments
}

func (reader *delimReader) IsEOF() bool {
	return reader.eof
}

func (reader *delimReader) ResetWith(dr delimReader) {
	reader.Reader.Reset(nil)
	reader.delim = dr.delim
}

func (reader *delimReader) Reset() {
	reader.Reader.Reset(nil)
	reader.n = 0
	reader.lastReadN = 0
	reader.segments = 0
	reader.eof = false
}

func (reader *delimReader) ReadOneBytes() (str []byte, err error) {
	if reader.eof {
		err = io.EOF
		return str, err
	}

	var rawLine []byte

	rawLine, err = reader.Reader.ReadSlice(reader.delim)
	reader.lastReadN = len(rawLine)
	reader.n += int64(reader.lastReadN)

	if err == io.EOF {
		reader.eof = true
		err = nil
	} else if err != nil {
		err = errors.Wrap(err)
		return str, err
	}

	str = bytes.TrimSuffix(rawLine, []byte{reader.delim})

	reader.segments++

	return str, err
}

// Not safe for parallel use
func (reader *delimReader) ReadOneString() (str string, err error) {
	if reader.eof {
		err = io.EOF
		return str, err
	}

	var rawLine string

	rawLine, err = reader.Reader.ReadString(reader.delim)
	reader.lastReadN = len(rawLine)
	reader.n += int64(reader.lastReadN)

	if err == io.EOF {
		reader.eof = true
		err = nil
	} else if err != nil {
		err = errors.Wrap(err)
		return str, err
	}

	str = strings.TrimSuffix(rawLine, string([]byte{reader.delim}))

	reader.segments++

	return str, err
}

// Not safe for parallel use
func (reader *delimReader) ReadOneKeyValue(
	sep string,
) (key, val string, err error) {
	if reader.eof {
		err = io.EOF
		return key, val, err
	}

	str, err := reader.ReadOneString()
	if err != nil {
		if err == io.EOF {
			reader.eof = true
		} else {
			err = errors.Wrap(err)
		}

		return key, val, err
	}

	loc := strings.Index(str, sep)

	if loc == -1 {
		err = errors.ErrorWithStackf(
			"expected at least one %q, but found none: %q",
			sep,
			str,
		)
		return key, val, err
	}

	key = str[:loc]
	val = str[loc+1:]

	return key, val, err
}

func (reader *delimReader) ReadOneKeyValueBytes(
	sep byte,
) (key, val []byte, err error) {
	if reader.eof {
		err = io.EOF
		return key, val, err
	}

	str, err := reader.ReadOneBytes()
	if err != nil {
		if err == io.EOF {
			reader.eof = true
		} else {
			err = errors.Wrap(err)
		}

		return key, val, err
	}

	loc := bytes.IndexByte(str, sep)

	if loc == -1 {
		err = errors.ErrorWithStackf(
			"expected at least one %q, but found none: %q",
			sep,
			str,
		)
		return key, val, err
	}

	key = str[:loc]
	val = str[loc+1:]

	return key, val, err
}
