package script_value

import (
	"io"
	"os"
	"os/exec"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
	"code.linenisgreat.com/dodder/go/lib/charlie/files"
)

type ScriptValue struct {
	script string
	cmd    *exec.Cmd
	file   *os.File
}

func (s ScriptValue) String() string {
	return s.script
}

func (s ScriptValue) IsEmpty() bool {
	return s.script == ""
}

func (s *ScriptValue) Set(v string) (err error) {
	s.script = v

	return err
}

func (s ScriptValue) Cmd() *exec.Cmd {
	return s.cmd
}

func (s *ScriptValue) RunWithInput() (w io.WriteCloser, r io.Reader, err error) {
	if s.IsEmpty() {
		err = errors.ErrorWithStackf("empty script")
		return w, r, err
	}

	s.cmd = exec.Command(s.script)

	if w, err = s.cmd.StdinPipe(); err != nil {
		errors.Wrap(err)
		return w, r, err
	}

	if r, err = s.cmd.StdoutPipe(); err != nil {
		errors.Wrap(err)
		return w, r, err
	}

	return w, r, err
}

func (s *ScriptValue) Run(input string) (r io.Reader, err error) {
	if s.IsEmpty() {
		if input == "" || input == "-" {
			r = os.Stdin
		} else {
			if s.file, err = files.Open(input); err != nil {
				err = errors.Wrap(err)
				return r, err
			}

			r = s.file
		}

		return r, err
	}

	if input == "" || input == "-" {
		s.cmd = exec.Command(s.script)
		s.cmd.Stdin = os.Stdin
	} else {
		s.cmd = exec.Command(s.script, input)
	}

	if r, err = s.cmd.StdoutPipe(); err != nil {
		errors.Wrap(err)
		return r, err
	}

	ui.Log().Print("starting")
	s.cmd.Start()

	return r, err
}

func (s *ScriptValue) Close() (err error) {
	ui.Log().Print("closing script")
	defer ui.Log().Print("done closing script")

	if s.file != nil {
		ui.Log().Print("closing file")
		err = files.Close(s.file)
	}

	if s.cmd != nil {
		ui.Log().Print("waiting for script")
		err = s.cmd.Wait()
	}

	return err
}
