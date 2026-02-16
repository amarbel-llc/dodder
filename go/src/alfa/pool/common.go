package pool

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"hash"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/src/_/pool_value"
)

var (
	bufioReader   = Make[bufio.Reader](nil, nil)
	bufioWriter   = Make[bufio.Writer](nil, nil)
	byteReaders   = Make[bytes.Reader](nil, nil)
	stringReaders = Make[strings.Reader](nil, nil)
	sha256Hash    = pool_value.Make(
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
) (stringReader *strings.Reader, repool func()) {
	stringReader = stringReaders.get()
	stringReader.Reset(value)

	repool = func() {
		stringReaders.put(stringReader)
	}

	return stringReader, repool
}

func GetByteReader(
	value []byte,
) (byteReader *bytes.Reader, repool func()) {
	byteReader = byteReaders.get()
	byteReader.Reset(value)

	repool = func() {
		byteReaders.put(byteReader)
	}

	return byteReader, repool
}

func GetSha256Hash() (hash hash.Hash, repool func()) {
	hash = sha256Hash.Get()

	repool = func() {
		sha256Hash.Put(hash)
	}

	return hash, repool
}

func GetBufferedWriter(
	writer io.Writer,
) (bufferedWriter *bufio.Writer, repool func()) {
	bufferedWriter = bufioWriter.get()
	bufferedWriter.Reset(writer)

	repool = func() {
		bufioWriter.put(bufferedWriter)
	}

	return bufferedWriter, repool
}

func GetBufferedReader(
	reader io.Reader,
) (bufferedReader *bufio.Reader, repool func()) {
	bufferedReader = bufioReader.get()
	bufferedReader.Reset(reader)

	repool = func() {
		bufioReader.put(bufferedReader)
	}

	return bufferedReader, repool
}
