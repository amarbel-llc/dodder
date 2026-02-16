package pool

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"hash"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
)

var (
	bufioReader   = Make[bufio.Reader](nil, nil)
	bufioWriter   = Make[bufio.Writer](nil, nil)
	byteReaders   = Make[bytes.Reader](nil, nil)
	stringReaders = Make[strings.Reader](nil, nil)
	sha256Hash    = MakeValue(
		func() hash.Hash {
			return sha256.New()
		},
		func(hash hash.Hash) {
			hash.Reset()
		},
	)
)

func GetStringReader(
	value string,
) (stringReader *strings.Reader, repool interfaces.FuncRepool) {
	stringReader, repool = stringReaders.GetWithRepool()
	stringReader.Reset(value)
	return stringReader, repool
}

func GetByteReader(
	value []byte,
) (byteReader *bytes.Reader, repool interfaces.FuncRepool) {
	byteReader, repool = byteReaders.GetWithRepool()
	byteReader.Reset(value)
	return byteReader, repool
}

func GetSha256Hash() (hash hash.Hash, repool interfaces.FuncRepool) {
	hash, repool = sha256Hash.GetWithRepool()
	return hash, repool
}

func GetBufferedWriter(
	writer io.Writer,
) (bufferedWriter *bufio.Writer, repool interfaces.FuncRepool) {
	bufferedWriter, repool = bufioWriter.GetWithRepool()
	bufferedWriter.Reset(writer)
	return bufferedWriter, repool
}

func GetBufferedReader(
	reader io.Reader,
) (bufferedReader *bufio.Reader, repool interfaces.FuncRepool) {
	bufferedReader, repool = bufioReader.GetWithRepool()
	bufferedReader.Reset(reader)
	return bufferedReader, repool
}
