package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/lima/store_fs"
	"code.linenisgreat.com/madder/go/pkgs/fd"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

func (local *Repo) DeleteFiles(fs interfaces.Collection[*fd.FD]) (err error) {
	deleteOp := store_fs.DeleteCheckout{}

	if err = deleteOp.Run(
		local.GetConfig().IsDryRun(),
		local.GetEnvWorkspace().GetStoreFS().GetFsOps(),
		local.PrinterFDDeleted(),
		fs,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
