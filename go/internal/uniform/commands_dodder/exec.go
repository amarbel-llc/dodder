package commands_dodder

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/sierra/repo_actions"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	dodder_exec "code.linenisgreat.com/dodder/go/lib/0/exec"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

func init() {
	utility.AddCmd("exec", &Exec{})
}

type Exec struct {
	command_components_dodder.LocalWorkingCopy
}

var _ command.CommandWithArgs = (*Exec)(nil)

func (cmd *Exec) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{
			{
				Name:        "object-id",
				Description: "object containing the script to execute",
				Required:    true,
			},
			{
				Name:        "args",
				Description: "arguments passed to the script",
				Variadic:    true,
			},
		},
	}}
}

func (cmd Exec) GetDescription() command.Description {
	return command.Description{
		Short: "execute a script stored as a blob",
	}
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
		execLuaOp := repo_actions.MakeExecLua(localWorkingCopy)

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

		if ar, err = u.GetEnvRepo().GetReadBlobStore().MakeBlobReader(
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
	cmd.Env = dodder_exec.MergeOSEnvWithAdder(u.GetEnvRepo())

	if err = cmd.Run(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
