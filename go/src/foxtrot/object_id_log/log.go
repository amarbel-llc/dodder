package object_id_log

import (
	"bufio"
	"os"
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	pool "code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
	"code.linenisgreat.com/dodder/go/src/charlie/files"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/foxtrot/triple_hyphen_io"
)

type Log struct {
	Path string
}

func (l Log) AppendEntry(entry Entry) (err error) {
	var file *os.File

	if file, err = files.OpenFile(
		l.Path,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0o666,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	typedBlob := &triple_hyphen_io.TypedBlob[Entry]{
		Type: ids.GetOrPanic(ids.TypeObjectIdLogVCurrent).TypeStruct,
		Blob: entry,
	}

	if _, err = Coder.EncodeTo(typedBlob, file); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (l Log) ReadAllEntries() (entries []Entry, err error) {
	var file *os.File

	if file, err = files.Open(l.Path); err != nil {
		if errors.IsNotExist(err) {
			err = nil
			return entries, err
		}

		err = errors.Wrap(err)
		return entries, err
	}

	defer errors.DeferredCloser(&err, file)

	bufferedReader, repoolBufferedReader := pool.GetBufferedReader(file)
	defer repoolBufferedReader()

	segments, err := segmentEntries(bufferedReader)
	if err != nil {
		err = errors.Wrap(err)
		return entries, err
	}

	for _, segment := range segments {
		var typedBlob triple_hyphen_io.TypedBlob[Entry]

		stringReader, repoolStringReader := pool.GetStringReader(segment)
		defer repoolStringReader()

		if _, err = Coder.DecodeFrom(
			&typedBlob,
			stringReader,
		); err != nil {
			err = errors.Wrap(err)
			return entries, err
		}

		entries = append(entries, typedBlob.Blob)
	}

	return entries, err
}

func segmentEntries(
	reader *bufio.Reader,
) (segments []string, err error) {
	var current strings.Builder
	boundaryCount := 0

	for line, errIter := range ohio.MakeLineSeqFromReader(reader) {
		if errIter != nil {
			err = errIter
			return segments, err
		}

		trimmed := strings.TrimSuffix(line, "\n")

		if trimmed == triple_hyphen_io.Boundary {
			boundaryCount++

			if boundaryCount > 2 && boundaryCount%2 == 1 {
				segments = append(segments, current.String())
				current.Reset()
			}
		}

		current.WriteString(line)
	}

	if current.Len() > 0 {
		segments = append(segments, current.String())
	}

	return segments, err
}
