package store_fs

import (
	"io"
	"os"
	"os/exec"

	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

// TODO include blobs
func (store *Store) runDiff3(
	local, base, remote *sku.FSItem,
) (merged *sku.FSItem, err error) {
	baseObjectPath := "/dev/null"

	if base.FDs.Len() > 0 {
		baseObjectPath = base.Object.GetPath()
	}

	cmd := exec.Command(
		"git",
		"merge-file",
		"-p",
		"-L=local",
		"-L=base",
		"-L=remote",
		local.Object.GetPath(),
		baseObjectPath,
		remote.Object.GetPath(),
	)

	cmd.Env = append(
		os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
	)

	var out io.ReadCloser

	if out, err = cmd.StdoutPipe(); err != nil {
		err = errors.Wrap(err)
		return merged, err
	}

	cmd.Stderr = os.Stderr

	var f *os.File

	if f, err = store.envRepo.GetTempLocal().FileTemp(); err != nil {
		err = errors.Wrap(err)
		return merged, err
	}

	defer errors.DeferredCloser(&err, f)

	if err = cmd.Start(); err != nil {
		err = errors.Wrap(err)
		return merged, err
	}

	if _, err = io.Copy(f, out); err != nil {
		err = errors.Wrap(err)
		return merged, err
	}

	merged = &sku.FSItem{}
	merged.ResetWith(local)
	merged.Object.Reset()
	merged.Blob.Reset()
	merged.FDs.Reset()

	hasConflict := false

	if err = cmd.Wait(); err != nil {
		var errExit *exec.ExitError

		if !errors.As(err, &errExit) {
			err = errors.Wrap(err)
			return merged, err
		}

		hasConflict = true
	}

	if err = merged.Object.SetPath(f.Name()); err != nil {
		err = errors.Wrap(err)
		return merged, err
	}

	if hasConflict {
		err = errors.Wrap(sku.MakeErrMergeConflict(merged))
	}

	return merged, err
}
