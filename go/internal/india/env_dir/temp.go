package env_dir

import (
	"os"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

// TODO only call reset temp when actually not resetting temp
func (env env) resetTempOnExit(ctx interfaces.ActiveContext) (err error) {
	errIn := ctx.Cause()

	if errIn != nil || env.debugOptions.NoTempDirCleanup {
		// ui.Err().Printf("temp dir: %q", s.TempLocal.BasePath)
	} else {
		if err = os.RemoveAll(env.GetTempLocal().BasePath); err != nil {
			err = errors.Wrapf(err, "failed to remove temp local")
			return err
		}
	}

	return err
}

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
