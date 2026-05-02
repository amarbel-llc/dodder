package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/charlie/fd"
	"code.linenisgreat.com/dodder/go/internal/mike/store_fs"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
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
