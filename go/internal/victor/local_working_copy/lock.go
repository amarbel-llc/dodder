package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
)

func (local *Repo) Lock() (err error) {
	if err = local.envRepo.GetLockSmith().Lock(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// TODO print organize files that were created if dry run or make it possible to
// commit dry-run transactions
func (local *Repo) Unlock() (err error) {
	ptl := local.PrinterTransacted()

	if local.storesInitialized {
		ui.Log().Printf(
			"konfig has changes: %t",
			local.GetConfigStore().HasChanges(),
		)
		ui.Log().Printf(
			"dormant has changes: %t",
			local.GetDormantIndex().HasChanges(),
		)

		var changes []string
		changes = append(changes, local.GetConfigStore().GetChanges()...)
		changes = append(changes, local.GetDormantIndex().GetChanges()...)
		local.GetStore().GetStreamIndex().SetNeedsFlushHistory(changes)

		ui.Log().Print("will flush inventory list")
		if err = local.store.FlushInventoryList(ptl); err != nil {
			err = errors.Wrap(err)
			return err
		}

		ui.Log().Print("will flush store")
		if err = local.store.Flush(
			local.PrinterHeader(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		ui.Log().Print("will flush konfig")
		if err = local.config.Flush(
			local.GetEnvRepo(),
			local.GetStore().GetTypedBlobStore(),
			local.PrinterHeader(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		ui.Log().Print("will flush dormant")
		if err = local.dormantIndex.Flush(
			local.GetEnvRepo(),
			local.PrinterHeader(),
			local.config.GetConfig().IsDryRun(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		local.GetStore().GetStreamIndex().SetNeedsFlushHistory(changes)

		wg := errors.MakeWaitGroupParallel()
		wg.Do(
			func() error {
				ui.Log().Print("will flush store second time")
				// second store flush is necessary because of konfig changes
				return local.store.Flush(
					local.PrinterHeader(),
				)
			},
		)

		if err = wg.GetError(); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	// explicitly do not unlock if there was an error to encourage user
	// interaction
	// and manual recovery
	if err = local.envRepo.GetLockSmith().Unlock(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
