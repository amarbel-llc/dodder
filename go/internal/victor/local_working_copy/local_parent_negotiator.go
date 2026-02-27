package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/golf/objects"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/tango/repo"
)

type ParentNegotiatorFirstAncestor struct {
	Local, Remote repo.Repo
}

func (parentNegotiator ParentNegotiatorFirstAncestor) GetParentNegotiator() sku.ParentNegotiator {
	return parentNegotiator
}

func (parentNegotiator ParentNegotiatorFirstAncestor) FindBestCommonAncestor(
	conflicted sku.Conflicted,
) (ancestor *sku.Transacted, err error) {
	var ancestorsLocal, ancestorsRemote []*sku.Transacted

	wg := errors.MakeWaitGroupParallel()

	wg.Do(
		func() (err error) {
			if ancestorsLocal, err = parentNegotiator.Local.ReadObjectHistory(
				conflicted.Local.GetObjectId(),
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		},
	)

	wg.Do(
		func() (err error) {
			if ancestorsRemote, err = parentNegotiator.Remote.ReadObjectHistory(
				conflicted.Local.GetObjectId(),
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		},
	)

	if err = wg.GetError(); err != nil {
		err = errors.Wrap(err)
		return ancestor, err
	}

	if len(ancestorsLocal) == 0 || len(ancestorsRemote) == 0 {
		return ancestor, err
	}

	// TODO repool all skus except ancestor

	ancestorLocal := ancestorsLocal[len(ancestorsLocal)-1]
	ancestorRemote := ancestorsRemote[len(ancestorsRemote)-1]

	if objects.EqualerSansTai.Equals(
		ancestorLocal.GetMetadata(),
		ancestorRemote.GetMetadata(),
	) {
		ancestor = ancestorLocal
	}

	return ancestor, err
}
