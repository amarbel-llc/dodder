package object_id_log

import (
	"bufio"
	"io"
	"os"
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/files"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/foxtrot/triple_hyphen_io"
)

func AppendEntry(
	path string,
	entry Entry,
) (err error) {
	var file *os.File

	if file, err = files.OpenFile(
		path,
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

func ReadAllEntries(
	path string,
) (entries []Entry, err error) {
	var file *os.File

	if file, err = files.Open(path); err != nil {
		if errors.IsNotExist(err) {
			err = nil
			return entries, err
		}

		err = errors.Wrap(err)
		return entries, err
	}

	defer errors.DeferredCloser(&err, file)

	segments, err := segmentEntries(bufio.NewReader(file))
	if err != nil {
		err = errors.Wrap(err)
		return entries, err
	}

	for _, segment := range segments {
		var typedBlob triple_hyphen_io.TypedBlob[Entry]

		if _, err = Coder.DecodeFrom(
			&typedBlob,
			strings.NewReader(segment),
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

	for {
		var line string

		line, err = reader.ReadString('\n')

		if err != nil && err != io.EOF {
			err = errors.Wrap(err)
			return segments, err
		}

		isEOF := errors.IsEOF(err)

		if isEOF && line == "" {
			if current.Len() > 0 {
				segments = append(segments, current.String())
			}

			err = nil

			break
		}

		trimmed := strings.TrimSuffix(line, "\n")

		if trimmed == triple_hyphen_io.Boundary {
			boundaryCount++

			// A new entry starts at boundary counts 1, 3, 5, ...
			// (odd boundaries are the opening --- of metadata).
			// The second boundary in each entry is boundary counts 2, 4, 6, ...
			// A new entry starts when we see an odd boundary after the first
			// entry.
			if boundaryCount > 2 && boundaryCount%2 == 1 {
				segments = append(segments, current.String())
				current.Reset()
			}
		}

		current.WriteString(line)

		if isEOF {
			if current.Len() > 0 {
				segments = append(segments, current.String())
			}

			err = nil

			break
		}
	}

	return segments, err
}
