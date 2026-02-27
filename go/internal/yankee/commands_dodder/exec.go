package commands_dodder

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/victor/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/whiskey/user_ops"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
)

func init() {
	utility.AddCmd("exec", &Exec{})
}

type Exec struct {
	command_components_dodder.LocalWorkingCopy
}

func (cmd Exec) Run(dep command.Request) {
	args := dep.PopArgs()

	if len(args) == 0 {
		errors.ContextCancelWithBadRequestf(dep, "needs at least Sku and possibly function name")
	}

	localWorkingCopy := cmd.MakeLocalWorkingCopy(dep)

	k, args := args[0], args[1:]

	var sk *sku.Transacted

	{
		var err error

		if sk, err = localWorkingCopy.GetEnvLua().GetSkuFromString(k); err != nil {
			localWorkingCopy.Cancel(err)
		}
	}

	switch {
	case strings.HasPrefix(sk.GetType().String(), "bash"):
		if err := cmd.runBash(localWorkingCopy, sk, args...); err != nil {
			localWorkingCopy.Cancel(err)
		}

	case strings.HasPrefix(sk.GetType().String(), "lua"):
		execLuaOp := user_ops.ExecLua{
			Repo: localWorkingCopy,
		}

		if err := execLuaOp.Run(sk, args...); err != nil {
			localWorkingCopy.Cancel(err)
		}
	}
}

func (c Exec) runBash(
	u *local_working_copy.Repo,
	tz *sku.Transacted,
	args ...string,
) (err error) {
	var scriptPath string

	func() {
		var ar io.ReadCloser

		if ar, err = u.GetEnvRepo().GetDefaultBlobStore().MakeBlobReader(
			tz.GetBlobDigest(),
		); err != nil {
			err = errors.Wrap(err)
			return
		}

		var f *os.File

		if f, err = u.GetEnvRepo().GetTempLocal().FileTemp(); err != nil {
			err = errors.Wrap(err)
			return
		}

		scriptPath = f.Name()

		defer errors.DeferredCloser(&err, f)

		if _, err = io.Copy(f, ar); err != nil {
			err = errors.Wrap(err)
			return
		}
	}()

	cmd := exec.Command(
		"bash",
		append([]string{scriptPath}, args...)...,
	)

	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err = cmd.Run(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
