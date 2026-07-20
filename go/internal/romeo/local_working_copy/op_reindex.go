package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

func (local *Repo) Reindex(allowLockFailures bool) {
	local.Must(errors.MakeFuncContextFromFuncErr(local.Lock))
	local.Must(errors.MakeFuncContextFromFuncErr(local.config.Reset))

	var lockfileOptions sku.LockfileOptions

	if allowLockFailures {
		lockfileOptions = sku.LockfileOptions{
			AllowTypeFailure:              true,
			AllowTagFailure:               true,
			AllowReferencedObjectFailure:  true,
			AllowBlobReferenceTypeFailure: true,
		}
	}

	local.Must(func(ctx interfaces.ActiveContext) error {
		return local.GetStore().Reindex(ctx, lockfileOptions)
	})

	local.Must(errors.MakeFuncContextFromFuncErr(local.Unlock))
}
