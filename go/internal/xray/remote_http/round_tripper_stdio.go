package remote_http

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"syscall"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_ui"
	"code.linenisgreat.com/dodder/go/internal/kilo/env_repo"
	"code.linenisgreat.com/dodder/go/internal/sierra/store_config"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/delim_io"
)

type RoundTripperStdio struct {
	exec.Cmd
	io.WriteCloser
	io.ReadCloser
	RoundTripperBufioWrappedSigner
}

func (roundTripper *RoundTripperStdio) InitializeWithLocal(
	envRepo env_repo.Env,
	config store_config.Config,
	pubkey domain_interfaces.MarklId,
) (err error) {
	roundTripper.PublicKey = pubkey

	// TODO design a better way of selecting binaries (factoring in zit /
	// dodder)
	roundTripper.Path = envRepo.GetExecPath()

	// TODO set first arg based on roundTripper.Path
	roundTripper.Args = []string{
		"dodder",
		"serve",
	}

	cliFlags := config.GetCLIFlags()

	roundTripper.Args = append(
		roundTripper.Args,
		cliFlags...,
	)

	roundTripper.Args = append(roundTripper.Args, "-")

	if err = roundTripper.initialize(envRepo); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (roundTripper *RoundTripperStdio) InitializeWithSSH(
	envUI env_ui.Env,
	arg string,
) (err error) {
	if roundTripper.Path, err = exec.LookPath("ssh"); err != nil {
		err = errors.Wrap(err)
		return err
	}

	roundTripper.Args = []string{
		"ssh",
		arg,
		"dodder",
		"serve",
	}

	if envUI.GetCLIConfig().GetVerbose() {
		roundTripper.Args = append(roundTripper.Args, "-verbose")
	}

	roundTripper.Args = append(roundTripper.Args, "-")

	if err = roundTripper.initialize(envUI); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (roundTripper *RoundTripperStdio) initialize(
	envUI env_ui.Env,
) (err error) {
	// roundTripper.Stderr = os.Stderr
	var stderrReadCloser io.ReadCloser

	if stderrReadCloser, err = roundTripper.StderrPipe(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	remotePrinterDone := make(chan struct{})

	// TODO refactor this into something in the ui package that closes
	// automatically at the end of a context
	go func() (err error) {
		defer close(remotePrinterDone)

		if _, err = delim_io.CopyWithPrefixOnDelim(
			'\n',
			"(remote) ",
			envUI.GetUI(),
			stderrReadCloser,
			false,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}()

	if roundTripper.WriteCloser, err = roundTripper.StdinPipe(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	roundTripper.Writer = bufio.NewWriter(roundTripper.WriteCloser)

	if roundTripper.ReadCloser, err = roundTripper.StdoutPipe(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	roundTripper.Reader = bufio.NewReader(roundTripper.ReadCloser)

	ui.Log().Printf("starting server: %q", roundTripper.Cmd.String())

	if err = roundTripper.Start(); err != nil {
		err = errors.Wrapf(err, "%#v", roundTripper.Cmd)
		return err
	}

	envUI.After(
		errors.MakeFuncContextFromFuncErr(
			roundTripper.makeCancel(remotePrinterDone),
		),
	)

	return err
}

func (roundTripper *RoundTripperStdio) makeCancel(
	readsDone <-chan struct{},
) errors.FuncErr {
	return func() error {
		return roundTripper.cancel(readsDone)
	}
}

func (roundTripper *RoundTripperStdio) cancel(
	readsDone <-chan struct{},
) (err error) {
	if roundTripper.Process != nil {
		if err = roundTripper.WriteCloser.Close(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = roundTripper.Process.Signal(syscall.SIGHUP); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	<-readsDone

	if err = roundTripper.Wait(); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			err = nil
		} else {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}
