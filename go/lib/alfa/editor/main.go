package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	dodder_exec "code.linenisgreat.com/dodder/go/lib/0/exec"
	"code.linenisgreat.com/dodder/go/lib/0/primordial"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/files"
	"github.com/google/shlex"
)

type Editor struct {
	utility string
	path    string
	name    string
	tipe    Type

	options []string

	ui       interfaces.FuncIter[string]
	envAdder interfaces.EnvVarsAdder
}

func getEditorUtility() string {
	var editor string

	if editor = os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	if editor = os.Getenv("VISUAL"); editor != "" {
		return editor
	}

	return "vim"
}

func MakeEditorWithVimOptions(
	funcUI interfaces.FuncIter[string],
	envAdder interfaces.EnvVarsAdder,
	options []string,
) (Editor, error) {
	return MakeEditor(
		funcUI,
		envAdder,
		map[Type][]string{
			TypeVim: options,
		},
	)
}

func MakeEditor(
	funcUI interfaces.FuncIter[string],
	envAdder interfaces.EnvVarsAdder,
	options map[Type][]string,
) (editor Editor, err error) {
	editor.utility = getEditorUtility()
	editor.ui = funcUI
	editor.envAdder = envAdder

	var utility []string

	if utility, err = shlex.Split(editor.utility); err != nil {
		err = errors.Wrap(err)
		return editor, err
	}

	if len(utility) < 1 {
		err = errors.ErrorWithStackf(
			"utility has no valid path: %q",
			editor.utility,
		)
		return editor, err
	}

	editor.path = utility[0]
	editor.options = append(editor.options, utility[1:]...)

	editor.name = filepath.Base(editor.path)

	switch editor.name {
	// TODO support other mechanisms of inferring vim as the editor
	case "vim", "nvim":
		editor.tipe = TypeVim
		editor.options = append(editor.options, "-f")
	}

	editor.options = append(editor.options, options[editor.tipe]...)

	return editor, err
}

func (editor Editor) Run(
	files []string,
) (err error) {
	if err = editor.ui(fmt.Sprintf("editor (%s) started", editor.name)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = editor.openWithArgs(files...); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = editor.ui(fmt.Sprintf("editor (%s) closed", editor.name)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (editor Editor) openWithArgs(fs ...string) (err error) {
	if len(fs) == 0 {
		err = errors.Wrap(files.ErrEmptyFileList)
		return err
	}

	allArgs := append(editor.options, fs...)

	cmd := exec.Command(
		editor.path,
		allArgs...,
	)

	if primordial.IsTty(os.Stdin) {
		cmd.Stdin = os.Stdin
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = dodder_exec.MergeOSEnvWithAdder(editor.envAdder)

	if err = cmd.Run(); err != nil {
		err = errors.Wrapf(err, "Cmd: %s", cmd)
		return err
	}

	return err
}
