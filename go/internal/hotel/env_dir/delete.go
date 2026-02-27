package env_dir

import (
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
)

func (env env) Delete(paths ...string) (err error) {
	for _, path := range paths {
		path = filepath.Clean(path)

		if path == "." {
			err = errors.ErrorWithStackf("invalid delete request: %q", path)
			return err
		}

		if env.IsDryRun() {
			ui.Err().Print("would delete:", path)
			return err
		}

		if err = os.RemoveAll(path); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}
