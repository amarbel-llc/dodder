package repo_actions

import (
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/echo/editor"
)

type OpenEditor struct {
	*local_working_copy.Repo
	VimOptions []string
}

func (c OpenEditor) Run(
	args ...string,
) (err error) {
	var e editor.Editor

	if e, err = editor.MakeEditorWithVimOptions(
		c.PrinterHeader(),
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
