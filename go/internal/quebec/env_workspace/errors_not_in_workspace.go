package env_workspace

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/hotel/workspace_config_blobs"
)

type ErrNotInWorkspace struct {
	*env
	offerToCreate bool
}

func (err ErrNotInWorkspace) Error() string {
	return "not in a workspace"
}

func (err ErrNotInWorkspace) Is(target error) bool {
	_, ok := target.(ErrNotInWorkspace)
	return ok
}

func (err ErrNotInWorkspace) ShouldShowStackTrace() bool {
	return false
}

func (err ErrNotInWorkspace) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func (err ErrNotInWorkspace) Recover(
	ctx errors.Context,
	retry interfaces.FuncRetry,
	abort interfaces.FuncRetryAborted,
) {
	if err.offerToCreate &&
		err.Confirm(
			"a workspace is necessary to run this command. create one?",
			"",
		) {
		blob := &workspace_config_blobs.V0{}

		if err := err.CreateWorkspace(blob); err != nil {
			ctx.Cancel(err)
		}

		retry()
	} else {
		abort(errors.Errorf("not creating a workspace"))
	}
}
