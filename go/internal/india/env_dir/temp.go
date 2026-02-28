package env_dir

import (
	"os"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
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
