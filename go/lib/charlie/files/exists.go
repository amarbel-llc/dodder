package files

import (
	"os"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

func Exists(path string) bool {
	_, err := os.Stat(path)
	return !errors.IsNotExist(err)
}

func AssertDir(path string) (err error) {
	fi, err := os.Stat(path)
	if err != nil {
		if errors.IsNotExist(err) {
			err = ErrNotDirectory(path)
		} else {
			err = errors.Wrap(err)
		}

		return err
	}

	if !fi.IsDir() {
		err = ErrNotDirectory(path)
		return err
	}

	return err
}
