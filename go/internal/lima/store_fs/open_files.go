package store_fs

import (
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/files"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
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
