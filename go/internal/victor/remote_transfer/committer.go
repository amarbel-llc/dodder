package remote_transfer

import (
	"code.linenisgreat.com/dodder/go/internal/kilo/env_repo"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/uniform/repo"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type committer struct {
	options     repo.ImporterOptions
	storeObject sku.RepoStore
	deduper     deduper
}

func (committer *committer) initialize(
	options repo.ImporterOptions,
	envRepo env_repo.Env,
	storeObject sku.RepoStore,
) {
	committer.options = options
	committer.storeObject = storeObject
	committer.deduper.initialize(options, envRepo)
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
