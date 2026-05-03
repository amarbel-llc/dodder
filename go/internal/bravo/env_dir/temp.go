package env_dir

import (
	"os"

	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/files"
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

type TemporaryFS = files.TemporaryFS
