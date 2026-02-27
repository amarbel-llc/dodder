package script_config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

type RemoteScript interface {
	Cmd(args ...string) (*exec.Cmd, error)
}

type RemoteScriptWithEnv interface {
	RemoteScript
	Environ() map[string]string
}

type writerTo struct {
	cmd *exec.Cmd
}

func MakeWriterTo(
	rs RemoteScript,
	env map[string]string,
	args ...string,
) (wt *writerTo, err error) {
	wt = &writerTo{}

	if rs == nil {
		err = errors.ErrorWithStackf("empty remote script")
		return wt, err
	}

	if wt.cmd, err = rs.Cmd(args...); err != nil {
		err = errors.Wrap(err)
		return wt, err
	}

	ui.Log().Print(wt.cmd)

	envCollapsed := os.Environ()

	for k, v := range env {
		envCollapsed = append(envCollapsed, fmt.Sprintf("%s=%s", k, v))
	}

	if rswe, ok := rs.(RemoteScriptWithEnv); ok {
		for k, v := range rswe.Environ() {
			envCollapsed = append(envCollapsed, fmt.Sprintf("%s=%s", k, v))
		}
	}

	ui.TodoP2("determine how stderr and env should be handled")
	// wt.cmd.Stderr = os.Stderr
	wt.cmd.Env = envCollapsed

	return wt, err
}

func MakeWriterToWithStdin(
	script RemoteScript,
	env map[string]string,
	reader io.Reader,
	args ...string,
) (writerTo *writerTo, err error) {
	if writerTo, err = MakeWriterTo(script, env, args...); err != nil {
		err = errors.Wrap(err)
		return writerTo, err
	}

	writerTo.cmd.Stdin = reader

	return writerTo, err
}

func (wt *writerTo) WriteTo(w io.Writer) (n int64, err error) {
	var pipeOut io.ReadCloser

	if pipeOut, err = wt.cmd.StdoutPipe(); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	var pipeErr io.ReadCloser

	if pipeErr, err = wt.cmd.StderrPipe(); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if err = wt.cmd.Start(); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	var bufErr bytes.Buffer
	chErrDone := make(chan struct{})

	go func() {
		io.Copy(&bufErr, pipeErr)
		close(chErrDone)
	}()

	if n, err = io.Copy(w, pipeOut); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	<-chErrDone

	if err = wt.cmd.Wait(); err != nil {
		var errExit *exec.ExitError

		if errors.As(err, &errExit) {
			errExit.Stderr = bufErr.Bytes()
		}

		err = errors.Wrapf(err, "Command: '%s'", wt.cmd.String())

		return n, err
	}

	return n, err
}
