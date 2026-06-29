package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
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

	// Pick the most recent common ancestor: the highest-TAI version present in
	// both histories. Versions are matched across repos by TAI, which is
	// preserved on transfer and — unlike the content locks that EqualerSansTai
	// compares — is independent of the repo pubkey. The same logical version
	// therefore still matches after the parent re-signs it under its own key
	// (the cross-pubkey case in #298).
	//
	// The previous code compared only the single oldest version of each
	// history (by content, including the pubkey-bearing type lock) and required
	// them to be equal. For a clean, linear fast-forward — where the parent
	// holds an older-but-on-path ancestor of the local head — that selected the
	// chain root, or across pubkeys no base at all, as the merge base. An empty
	// or too-old base makes the parent's own progression look like a divergent
	// change and manufactures a false "merging required" conflict (#298).
	// Selecting the newest shared version makes the parent's head the base, so
	// the local head merges as a fast-forward; genuinely divergent histories
	// share no TAI and still conflict.
	for _, candidate := range ancestorsLocal {
		isCommon := false

		for _, remote := range ancestorsRemote {
			if candidate.GetTai().Equals(remote.GetTai()) {
				isCommon = true
				break
			}
		}

		if !isCommon {
			continue
		}

		if ancestor == nil || ancestor.GetTai().Less(candidate.GetTai()) {
			ancestor = candidate
		}
	}

	return ancestor, err
}
