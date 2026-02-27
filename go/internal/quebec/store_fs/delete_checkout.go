package store_fs

import (
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/golf/fd"
	"code.linenisgreat.com/dodder/go/internal/kilo/env_repo"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

type DeleteCheckout struct{}

func (c DeleteCheckout) Run(
	dryRun bool,
	s env_repo.Env,
	p interfaces.FuncIter[*fd.FD],
	fs interfaces.Collection[*fd.FD],
) (err error) {
	els := quiter.ElementsSorted(
		fs,
		func(i, j *fd.FD) bool {
			return i.String() < j.String()
		},
	)

	if dryRun {
		for _, f := range els {
			if err = p(f); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		return err
	}

	dirs := make([]string, 0)

	for _, fd := range els {
		path := fd.String()

		if path == "." {
			continue
		}

		if pRel, pErr := filepath.Rel(
			s.GetCwd(),
			fd.String(),
		); pErr == nil {
			path = pRel
		}

		func() {
			if fd.IsDir() && fd.GetPath() != s.GetCwd() {
				dirs = append(dirs, fd.GetPath())
				return
			}

			dir := filepath.Dir(path)

			if dir == s.GetCwd() {
				return
			}

			// Occurs when the file is top-level relative to Cwd, and so has no parent
			// directory. This is a false-positive and should be ignored.
			if dir == "." {
				return
			}

			dirs = append(dirs, dir)
		}()

		if err = s.Delete(path); err != nil {
			if errors.IsNotExist(err) {
				err = nil
			} else {
				err = errors.Wrapf(err, "FD: %s", fd)
				return err
			}
		}

		if p != nil {
			if err = p(fd); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}
	}

	for _, d := range dirs {
		var contents []os.DirEntry

		if contents, err = files.ReadDir(d); err != nil {
			if errors.IsNotExist(err) {
				err = nil
			} else {
				err = errors.Wrapf(err, "Dir: %s", d)
			}

			continue
		}

		if len(contents) != 0 {
			continue
		}

		if err = s.Delete(d); err != nil {
			err = errors.Wrap(err)
			return err
		}

		var f *fd.FD

		if f, err = fd.MakeFromDirPath(d); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if p != nil {
			if err = p(f); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}
	}

	return err
}
