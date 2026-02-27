package store_fs

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/files"
)

type OpenFiles struct{}

func (c OpenFiles) Run(
	ph interfaces.FuncIter[string],
	args ...string,
) (err error) {
	if len(args) == 0 {
		return err
	}

	if err = files.OpenFiles(args...); err != nil {
		err = errors.Wrapf(err, "%q", args)
		return err
	}

	v := "opening files"

	if err = ph(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
