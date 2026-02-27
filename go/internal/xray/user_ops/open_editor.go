package user_ops

import (
	"code.linenisgreat.com/dodder/go/internal/whiskey/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/echo/editor"
)

type OpenEditor struct {
	VimOptions []string
}

func (c OpenEditor) Run(
	u *local_working_copy.Repo,
	args ...string,
) (err error) {
	var e editor.Editor

	if e, err = editor.MakeEditorWithVimOptions(
		u.PrinterHeader(),
		c.VimOptions,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = e.Run(args); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
