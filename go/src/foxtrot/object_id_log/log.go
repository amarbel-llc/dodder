package object_id_log

import (
	"bufio"
	"os"

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

	bufferedReader := bufio.NewReader(file)

	for {
		var typedBlob triple_hyphen_io.TypedBlob[Entry]

		if _, err = Coder.DecodeFrom(&typedBlob, bufferedReader); err != nil {
			if errors.IsEOF(err) {
				err = nil
				break
			}

			err = errors.Wrap(err)
			return entries, err
		}

		entries = append(entries, typedBlob.Blob)
	}

	return entries, err
}
