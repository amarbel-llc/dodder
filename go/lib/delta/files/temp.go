package files

import (
	"os"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type TemporaryFS struct {
	BasePath string
}

func (fs TemporaryFS) DirTemp() (d string, err error) {
	return fs.DirTempWithTemplate("")
}

func (fs TemporaryFS) DirTempWithTemplate(
	template string,
) (dir string, err error) {
	if dir, err = os.MkdirTemp(fs.BasePath, template); err != nil {
		err = errors.Wrap(err)
		return dir, err
	}

	return dir, err
}

func (fs TemporaryFS) FileTemp() (file *os.File, err error) {
	if file, err = fs.FileTempWithTemplate(""); err != nil {
		err = errors.Wrap(err)
		return file, err
	}

	return file, err
}

func (fs TemporaryFS) FileTempWithTemplate(
	template string,
) (file *os.File, err error) {
	if file, err = os.CreateTemp(fs.BasePath, template); err != nil {
		err = errors.Wrap(err)
		return file, err
	}

	return file, err
}
