package repo_actions

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/editor"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

type OpenEditor struct {
	*repo
	VimOptions []string
}

func (c OpenEditor) Run(
	args ...string,
) (err error) {
	var e editor.Editor

	if e, err = editor.MakeEditorWithVimOptions(
		c.PrinterHeader(),
		c.GetEnvRepo(),
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
