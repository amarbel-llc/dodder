package store

import (
	"sort"

	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/file_lock"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/box_format"
	"code.linenisgreat.com/dodder/go/internal/golf/object_finalizer"
	"code.linenisgreat.com/dodder/go/internal/hotel/inventory_list_coders"
	"code.linenisgreat.com/dodder/go/internal/india/config_log"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

// konfigState captures the three scalar values needed to backfill a
// single historical config state into the config log. Pooled
// *sku.Transacted objects are never retained past their loop iteration
// (they are reused); only these detached values are.
type konfigState struct {
	blobDigest markl.Id
	configType string
	tai        ids.Tai
}

// TODO-P2 add support for quiet reindexing
func (store *Store) Reindex(
	context interfaces.ActiveContext,
	lockfileOptions sku.LockfileOptions,
) (err error) {
	if !store.GetEnvRepo().GetLockSmith().IsAcquired() {
		err = file_lock.ErrLockRequired{
			Operation: "reindex",
		}

		return err
	}

	if err = store.ResetIndexes(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.GetEnvRepo().ResetCache(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var reindexer sku.Reindexer

	if reindexer, err = store.streamIndex.MakeReindexer(context); err != nil {
		err = errors.Wrap(err)
		return err
	}

	commitFacilitator := commitFacilitator{
		Store: store,
		index: reindexer,
	}

	// Build the config log to decide whether to backfill it from the old
	// konfig object history. Backfill only when the log is absent
	// (un-migrated repo): an existing log already holds the config (and
	// possibly post-migration edits), so leave it untouched. This guard
	// makes reindex idempotent for config.
	cfgLog := config_log.Make(
		store.GetEnvRepo(),
		inventory_list_coders.MakeCloset(
			store.GetEnvRepo(),
			box_format.MakeBoxTransactedArchive(
				store.GetEnvRepo(),
				options_print.Options{}.WithPrintTai(true),
			),
		),
	)

	var backfillConfig bool

	{
		_, repoolHead, headErr := cfgLog.Head()

		if headErr != nil {
			if errors.Is(headErr, config_log.ErrEmpty) {
				backfillConfig = true
			} else {
				err = errors.Wrap(headErr)
				return err
			}
		} else {
			repoolHead()
		}
	}

	var konfigStates []konfigState

	type objectWithError struct {
		error
		sku.ObjectWithList
	}

	objectsWithErrors := make(map[string]objectWithError)
	unidentifiedErrors := make([]error, 0)

	seq := store.GetInventoryListStore().AllInventoryListObjectsAndContents()

	// TODO switch to reusing fsck command structure
	for objectWithList, iterErr := range seq {
		if iterErr != nil {
			if objectWithList.List == nil {
				unidentifiedErrors = append(unidentifiedErrors, iterErr)
			} else {
				keyBytes := objectWithList.List.GetObjectDigest().GetBytes()

				objectsWithErrors[string(keyBytes)] = objectWithError{
					error: iterErr,
					ObjectWithList: sku.ObjectWithList{
						List: func() *sku.Transacted { c, _ := objectWithList.List.CloneTransacted(); return c }(), //repool:owned
					},
				}
			}

			continue
		}

		if objectWithList.Object == nil {
			panic("empty object")
		}

		// Divert konfig objects from the stream-index rebuild: config is
		// no longer an indexed object (FDR 0020). When backfilling,
		// capture the scalar values needed to re-emit the state into the
		// config log, then skip reindexOne entirely.
		if objectWithList.Object.GetObjectId().GetGenre() == genres.Config {
			if backfillConfig {
				var state konfigState
				state.blobDigest.ResetWithMarklId(
					objectWithList.Object.GetBlobDigest(),
				)
				state.configType = objectWithList.Object.GetType().String()
				state.tai = objectWithList.Object.GetTai()

				konfigStates = append(konfigStates, state)
			}

			continue
		}

		if err = store.reindexOne(commitFacilitator, objectWithList, lockfileOptions); err != nil {
			keyBytes := objectWithList.List.GetObjectDigest().GetBytes()

			objectsWithErrors[string(keyBytes)] = objectWithError{
				error: err,
				ObjectWithList: sku.ObjectWithList{
					Object: func() *sku.Transacted { c, _ := objectWithList.Object.CloneTransacted(); return c }(), //repool:owned
					List:   func() *sku.Transacted { c, _ := objectWithList.List.CloneTransacted(); return c }(),   //repool:owned
				},
			}

			continue
		}
	}

	if backfillConfig && len(konfigStates) > 0 {
		if err = store.backfillConfigLog(cfgLog, konfigStates); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if len(unidentifiedErrors) > 0 {
		store.envRepo.GetUI().Print("unidentified errors:")

		for _, err := range unidentifiedErrors {
			ui.CLIErrorTreeEncoder.EncodeTo(err, store.envRepo.GetUI())
		}
	}

	if len(objectsWithErrors) > 0 {
		store.envRepo.GetUI().Print("objects with errors:")

		var lockFailureCount int

		for _, objectWithError := range objectsWithErrors {
			ui.CLIErrorTreeEncoder.EncodeTo(objectWithError.error, store.envRepo.GetUI())

			if errors.Is(
				objectWithError.error,
				object_finalizer.ErrFailedToReadCurrentLockObject,
			) {
				lockFailureCount++
			}

			if objectWithError.Object == nil {
				store.envRepo.GetUI().Printf(
					"Error: %s, List: %q",
					objectWithError.error,
					sku.String(objectWithError.List),
				)
			} else {
				store.envRepo.GetUI().Printf(
					"Error: %s, List: %q, Object: %q",
					objectWithError.error,
					sku.String(objectWithError.List),
					sku.String(objectWithError.Object),
				)
			}
		}

		// lockFailureCount can only be nonzero here when the caller did not
		// already set the corresponding LockfileOptions field — a tolerated
		// failure never reaches objectsWithErrors in the first place (see
		// object_finalizer.WriteLockfile). So a nonzero count always means
		// -allow_lock_failures would have suppressed at least one of the
		// errors just printed above.
		if lockFailureCount > 0 {
			store.envRepo.GetUI().Printf(
				"%d of the errors above are missing lock targets (a type, tag, or referenced object that no longer exists) — rerun with -allow_lock_failures to tolerate them and finish building the index anyway",
				lockFailureCount,
			)
		}
	}

	return err
}

// backfillConfigLog re-emits the historical konfig states into the
// config log oldest->newest. States are deduped by (tai, blobDigest)
// because AllInventoryListObjectsAndContents may yield the same konfig
// version across multiple inventory lists. Each Append re-signs and
// re-chains into a fresh mother-sig chain, preserving the original blob
// digests and tais.
func (store *Store) backfillConfigLog(
	cfgLog config_log.Log,
	states []konfigState,
) (err error) {
	seen := make(map[string]struct{}, len(states))
	deduped := make([]konfigState, 0, len(states))

	for _, state := range states {
		key := state.tai.String() + "\x00" + string(state.blobDigest.GetBytes())

		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		deduped = append(deduped, state)
	}

	// tai is sec.asec; sort oldest->newest so the log's append order
	// reproduces the original config history.
	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].tai.Before(deduped[j].tai)
	})

	for i := range deduped {
		state := &deduped[i]

		if err = cfgLog.Append(
			&state.blobDigest,
			ids.MustTypeStruct(state.configType),
			state.tai,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (store *Store) reindexOne(
	commitFacilitator commitFacilitator,
	object sku.ObjectWithList,
	lockfileOptions sku.LockfileOptions,
) (err error) {
	storeOptions := sku.GetStoreOptionsReindex()
	storeOptions.LockfileOptions = lockfileOptions

	options := sku.CommitOptions{
		StoreOptions: storeOptions,
	}

	if err = commitFacilitator.commit(object.Object, options); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.GetAbbrStore().AddObjectToIdIndex(
		object.Object,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
