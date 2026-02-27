package files

import (
	"bytes"
	"io/fs"
	"os/exec"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

//go:generate stringer -type=FileType
type FileType byte

const (
	FileTypeUnknown = FileType(iota)
	FileTypeText
	FileTypeExecutable
	FileTypeData
)

func GetFileTypeForPath(path string) (tipe FileType, err error) {
	cmd := exec.Command(
		"file",
		path,
	)

	var msg []byte

	if msg, err = cmd.CombinedOutput(); err != nil {
		if isNotExists(err, msg) {
			err = fs.ErrNotExist
		}

		return tipe, err
	}

	_, tail, ok := bytes.Cut(msg, []byte{':'})

	if !ok {
		err = errors.ErrorWithStackf("`file` output invalid, expected `:`: %s", msg)
		return tipe, err
	}

	switch {
	case bytes.Contains(tail, []byte("text")):
		tipe = FileTypeText

	case bytes.Contains(tail, []byte("executable")):
		tipe = FileTypeExecutable

	case bytes.Contains(tail, []byte("data")):
		tipe = FileTypeData

	default:
		// TODO consider not making this an error?
		err = errors.ErrorWithStackf("`file` output unknown: %s", msg)
		return tipe, err
	}

	return tipe, err
}
