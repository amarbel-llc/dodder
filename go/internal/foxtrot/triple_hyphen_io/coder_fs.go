package triple_hyphen_io

import (
	"bytes"
	"os"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/files"
)

func DecodeFromFileInto[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
](
	typedBlob *TypedBlob[BLOB],
	coders CoderToTypedBlob[BLOB],
	path string,
) (err error) {
	var file *os.File

	if path == "-" {
		file = os.Stdin
	} else {
		if file, err = files.OpenExclusiveReadOnly(
			path,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloser(&err, file)
	}

	if _, err = coders.DecodeFrom(typedBlob, file); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func DecodeFromFile[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
](
	coders CoderToTypedBlob[BLOB],
	path string,
) (typedBlob TypedBlob[BLOB], err error) {
	if err = DecodeFromFileInto(&typedBlob, coders, path); err != nil {
		err = errors.Wrap(err)
		return typedBlob, err
	}

	return typedBlob, err
}

// For decodes where typedBlob.Blob should be populated with an empty struct
// when the file is missing
func DecodeFromFileOrEmptyBuffer[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
](
	coders CoderToTypedBlob[BLOB],
	path string,
	permitNotExist bool,
) (typedBlob TypedBlob[BLOB], err error) {
	typedBlob, err = DecodeFromFile(coders, path)

	if err == nil {
		return typedBlob, err
	}

	if _, err = coders.DecodeFrom(&typedBlob, bytes.NewBuffer(nil)); err != nil {
		err = errors.Wrap(err)
		return typedBlob, err
	}

	return typedBlob, err
}

func EncodeToFile[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
](
	coders CoderToTypedBlob[BLOB],
	typedBlob *TypedBlob[BLOB],
	path string,
) (err error) {
	var file *os.File

	if path == "-" {
		file = os.Stdin
	} else {
		if file, err = files.CreateExclusiveWriteOnly(
			path,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloser(&err, file)
	}

	if _, err = coders.EncodeTo(typedBlob, file); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
