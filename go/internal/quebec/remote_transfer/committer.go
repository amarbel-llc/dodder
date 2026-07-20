package remote_transfer

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

type committer struct {
	options     repo.ImporterOptions
	storeObject sku.StoreCommitter
	deduper     deduper
}

func (committer *committer) initialize(
	options repo.ImporterOptions,
	storeObject sku.StoreCommitter,
) {
	committer.options = options
	committer.storeObject = storeObject
	committer.deduper.initialize(options)
}

func (committer *committer) Commit(
	object *sku.Transacted,
	commitOptions sku.CommitOptions,
) (err error) {
	if err = committer.deduper.shouldCommit(object); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = committer.storeObject.Commit(
		object,
		commitOptions,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}
	return err
}
