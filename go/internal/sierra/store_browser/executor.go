//go:build chrest

package store_browser

import (
	"net/url"

	"code.linenisgreat.com/dodder/go/internal/_/external_state"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type executor struct {
	store      *Store
	query      *queries.Query
	out        interfaces.FuncIter[sku.SkuType]
	checkedOut sku.CheckedOut
}

func (executor *executor) tryToEmitOneExplicitlyCheckedOut(
	internal *sku.Transacted,
	item Item,
) (err error) {
	executor.checkedOut.GetSkuExternal().GetObjectIdMutable().Reset()

	var uSku *url.URL

	if uSku, err = executor.store.getUrl(internal); err != nil {
		err = errors.Wrap(err)
		return err
	}

	sku.TransactedResetter.ResetWith(executor.checkedOut.GetSku(), internal)
	sku.TransactedResetter.ResetWith(executor.checkedOut.GetSkuExternal().GetSku(), internal)

	if *uSku == item.Url.Url() {
		// c.co.SetState(checked_out_state.ExistsAndSame)
	} else {
		// c.co.SetState(checked_out_state.Changed)
	}

	executor.checkedOut.GetSkuExternal().SetExternalState(external_state.Tracked)

	if err = executor.tryToEmitOneCommon(item); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (executor *executor) tryToEmitOneRecognized(
	internal *sku.Transacted,
	item Item,
) (err error) {
	executor.checkedOut.SetState(checked_out_state.Recognized)

	if !queries.ContainsSkuCheckedOutState(executor.query, executor.checkedOut.GetState()) {
		return err
	}

	sku.TransactedResetter.ResetWith(executor.checkedOut.GetSku(), internal)
	sku.TransactedResetter.ResetWith(executor.checkedOut.GetSkuExternal().GetSku(), internal)

	// if err = item.WriteToObjectId(&c.co.External.ObjectId); err != nil {
	// 	err = errors.Wrap(err)
	// 	return
	// }

	executor.checkedOut.SetState(checked_out_state.Recognized)
	executor.checkedOut.GetSkuExternal().SetExternalState(external_state.Recognized)

	if err = executor.tryToEmitOneCommon(item); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (executor *executor) tryToEmitOneUntracked(
	item Item,
) (err error) {
	executor.checkedOut.SetState(checked_out_state.Untracked)

	if !queries.ContainsSkuCheckedOutState(executor.query, executor.checkedOut.GetState()) {
		return err
	}

	sku.TransactedResetter.Reset(executor.checkedOut.GetSkuExternal().GetSku())
	sku.TransactedResetter.Reset(executor.checkedOut.GetSku())

	if err = executor.checkedOut.GetSkuExternal().GetMetadataMutable().GetDescriptionMutable().Set(
		item.Title,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = executor.tryToEmitOneCommon(item); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (executor *executor) tryToEmitOneCommon(
	item Item,
) (err error) {
	external := executor.checkedOut.GetSkuExternal()

	if err = item.WriteToExternal(external); err != nil {
		err = errors.Wrap(err)
		return err
	}

	external.GetObjectIdMutable().SetGenre(genres.Zettel)
	external.GetExternalObjectIdMutable().SetGenre(genres.Zettel)

	if !queries.ContainsExternalSku(executor.query, external, executor.checkedOut.GetState()) {
		return err
	}

	executor.checkedOut.GetSkuExternal().SetRepoId(executor.store.externalStoreInfo.RepoId)

	if err = executor.out(&executor.checkedOut); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
