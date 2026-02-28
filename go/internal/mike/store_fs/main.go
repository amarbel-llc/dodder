package store_fs

import (
	"encoding/gob"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/_/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/_/doddish"
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/bravo/file_extensions"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/fd"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/object_metadata_fmt_triple_hyphen"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/lima/store_workspace"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
	"code.linenisgreat.com/dodder/go/lib/delta/collections_value"
)

func init() {
	gob.Register(sku.Transacted{})
}

func Make(
	config sku.Config,
	deletedPrinter interfaces.FuncIter[*fd.FD],
	fileExtensions file_extensions.Config,
	envRepo env_repo.Env,
) (store *Store, err error) {
	blobStore := envRepo.GetDefaultBlobStore()

	store = &Store{
		config:         config,
		deletedPrinter: deletedPrinter,
		envRepo:        envRepo,
		fileEncoder:    MakeFileEncoder(envRepo, config),
		fileExtensions: fileExtensions,
		dir:            envRepo.GetCwd(),
		dirInfo: makeObjectsWithDir(
			fileExtensions,
			envRepo,
		),
		deleted: collections_value.MakeMutableValueSet[*fd.FD](
			nil,
		),
		deletedInternal: collections_value.MakeMutableValueSet[*fd.FD](
			nil,
		),
		metadataTextParser: object_metadata_fmt_triple_hyphen.Factory{
			EnvDir:    envRepo,
			BlobStore: blobStore,
		}.MakeTextParser(),
	}

	return store, err
}

type Store struct {
	config             sku.Config
	deletedPrinter     interfaces.FuncIter[*fd.FD]
	metadataTextParser object_metadata_fmt_triple_hyphen.Parser
	envRepo            env_repo.Env
	fileEncoder        FileEncoder
	inlineTypeChecker  ids.InlineTypeChecker
	fileExtensions     file_extensions.Config
	dir                string

	dirInfo

	deleteLock      sync.Mutex
	deleted         fd.MutableSet
	deletedInternal fd.MutableSet
}

func (store *Store) GetExternalStoreLike() store_workspace.StoreLike {
	return store
}

// Deletions of user objects that should be exposed to the user
func (store *Store) DeleteCheckedOut(co *sku.CheckedOut) (err error) {
	external := co.GetSkuExternal()

	var item *sku.FSItem

	if item, err = store.ReadFSItemFromExternal(external); err != nil {
		err = errors.Wrap(err)
		return err
	}

	store.deleteLock.Lock()
	defer store.deleteLock.Unlock()

	for fd := range item.FDs.All() {
		store.deleted.Add(fd)
	}

	return err
}

// Deletions of "transient" internal objects that should not be exposed to the
// user
func (store *Store) DeleteCheckedOutInternal(co *sku.CheckedOut) (err error) {
	external := co.GetSkuExternal()

	var i *sku.FSItem

	if i, err = store.ReadFSItemFromExternal(external); err != nil {
		err = errors.Wrap(err)
		return err
	}

	store.deleteLock.Lock()
	defer store.deleteLock.Unlock()

	for fd := range i.FDs.All() {
		store.deletedInternal.Add(fd)
	}

	return err
}

func (store *Store) Flush() (err error) {
	deleteOp := DeleteCheckout{}

	if err = deleteOp.Run(
		store.config.IsDryRun(),
		store.envRepo,
		store.deletedPrinter,
		store.deleted,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = deleteOp.Run(
		store.config.IsDryRun(),
		store.envRepo,
		nil,
		store.deletedInternal,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	store.deleted.Reset()
	store.deletedInternal.Reset()

	return err
}

func (store *Store) String() (out string) {
	if quiter.Len(
		store.dirInfo.probablyCheckedOut,
		store.definitelyNotCheckedOut,
	) == 0 {
		return out
	}

	sb := &strings.Builder{}
	sb.WriteRune(doddish.OpGroupOpen.ToRune())

	hasOne := false

	writeOneIfNecessary := func(v interfaces.Stringer) (err error) {
		if hasOne {
			sb.WriteRune(doddish.OpOr.ToRune())
		}

		sb.WriteString(v.String())

		hasOne = true

		return err
	}

	for fsItem := range store.dirInfo.probablyCheckedOut.All() {
		writeOneIfNecessary(fsItem)
	}

	for fsItem := range store.definitelyNotCheckedOut.All() {
		writeOneIfNecessary(fsItem)
	}

	sb.WriteRune(doddish.OpGroupClose.ToRune())

	out = sb.String()
	return out
}

func (store *Store) GetExternalObjectIds() (fsItems []*sku.FSItem, err error) {
	if err = store.dirInfo.processRootDir(); err != nil {
		err = errors.Wrap(err)
		return fsItems, err
	}

	fsItems = make([]*sku.FSItem, 0)
	var lock sync.Mutex

	if err = store.All(
		func(kfp *sku.FSItem) (err error) {
			lock.Lock()
			defer lock.Unlock()

			fsItems = append(fsItems, kfp)

			return err
		},
	); err != nil {
		err = errors.Wrap(err)
		return fsItems, err
	}

	return fsItems, err
}

func (store *Store) GetFSItemsForDir(
	fd *fd.FD,
) (items []*sku.FSItem, err error) {
	if !fd.IsDir() {
		err = errors.ErrorWithStackf("not a directory: %q", fd)
		return items, err
	}

	if items, err = store.dirInfo.processDir(fd.GetPath()); err != nil {
		err = errors.Wrap(err)
		return items, err
	}

	return items, err
}

// TODO confirm against actual Object Id
func (store *Store) GetFSItemsForString(
	baseDir string,
	value string,
	tryPattern bool,
) (items []*sku.FSItem, err error) {
	if value == "." {
		if items, err = store.GetExternalObjectIds(); err != nil {
			err = errors.Wrap(err)
			return items, err
		}

		return items, err
	}

	var fdee *fd.FD

	if fdee, err = fd.MakeFromPath(
		baseDir,
		value,
		store.envRepo.GetDefaultBlobStore(),
	); err != nil {
		if errors.IsNotExist(err) && tryPattern {
			if items, err = store.dirInfo.processFDPattern(
				value,
				filepath.Join(store.dir, fmt.Sprintf("%s*", value)),
				store.dir,
			); err != nil {
				err = errors.Wrap(err)
				return items, err
			}
		} else {
			err = errors.Wrap(err)
		}

		return items, err
	}

	if fdee.IsDir() {
		if items, err = store.GetFSItemsForDir(fdee); err != nil {
			err = errors.Wrap(err)
			return items, err
		}
	} else {
		if _, items, err = store.dirInfo.processFD(fdee); err != nil {
			err = errors.Wrap(err)
			return items, err
		}
	}

	return items, err
}

func (store *Store) GetObjectIdsForString(
	queryLiteral string,
) (objectIds []sku.ExternalObjectId, err error) {
	var items []*sku.FSItem

	if items, err = store.GetFSItemsForString(
		store.root,
		queryLiteral,
		false,
	); err != nil {
		err = errors.Wrap(err)
		return objectIds, err
	}

	for _, item := range items {
		var eoid ids.ExternalObjectId

		if err = item.WriteToExternalObjectId(
			&eoid,
			store.envRepo,
		); err != nil {
			err = errors.Wrap(err)
			return objectIds, err
		}

		objectIds = append(objectIds, &eoid)
	}

	return objectIds, err
}

func (store *Store) Get(
	objectId ids.Id,
) (fsItem *sku.FSItem, ok bool) {
	return store.dirInfo.probablyCheckedOut.Get(objectId.String())
}

func (store *Store) Initialize(
	storeSupplies store_workspace.Supplies,
) (err error) {
	store.root = storeSupplies.WorkspaceDir
	store.storeSupplies = storeSupplies
	return err
}

func (store *Store) ReadAllExternalItems() (err error) {
	if err = store.dirInfo.processRootDir(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) ReadFSItemFromExternal(
	transactedGetter sku.TransactedGetter,
) (item *sku.FSItem, err error) {
	item = &sku.FSItem{} // TODO use pool or use dir_items?
	item.Reset()

	object := transactedGetter.GetSku()

	// TODO handle sort order
	for field := range object.GetMetadata().GetIndex().GetFields() {
		var fdee *fd.FD

		switch strings.ToLower(field.Key) {
		case "object":
			fdee = &item.Object

		case "blob":
			fdee = &item.Blob

		case "conflict":
			fdee = &item.Conflict

		case "lockfile":
			fdee = &item.Lockfile

		default:
			err = errors.ErrorWithStackf("unexpected field: %#v", field)
			return item, err
		}

		// if we've already set one of object, blob, or conflict, don't set it
		// again
		// and instead add a new FD to the item
		if !fdee.IsEmpty() {
			fdee = &fd.FD{}
		}

		if err = fdee.SetIgnoreNotExists(field.Value); err != nil {
			err = errors.Wrapf(err, "Key: %q", field.Key)
			return item, err
		}

		if err = item.FDs.Add(fdee); err != nil {
			err = errors.Wrap(err)
			return item, err
		}
	}

	if err = item.ExternalObjectId.SetObjectIdLike(
		&object.ObjectId,
	); err != nil {
		err = errors.Wrap(err)
		return item, err
	}

	// external.ObjectId.ResetWith(conflicted.GetSkuExternal().GetObjectId())
	// TODO populate FD
	if !object.ExternalObjectId.IsEmpty() {
		if err = item.ExternalObjectId.SetObjectIdLike(
			&item.ExternalObjectId,
		); err != nil {
			err = errors.Wrap(err)
			return item, err
		}
	}

	return item, err
}

func (store *Store) WriteFSItemToExternal(
	item *sku.FSItem,
	transactedGetter sku.TransactedGetter,
) (err error) {
	object := transactedGetter.GetSku()
	object.GetMetadataMutable().GetIndexMutable().GetFieldsMutable().Reset()

	metadata := object.GetMetadataMutable()
	metadata.GetTaiMutable().ResetWith(item.GetTai())

	var mode checkout_mode.Mode

	if mode, err = item.GetCheckoutModeOrError(); err != nil {
		if sku.IsErrMergeConflict(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
			return err
		}
	}

	switch {
	case mode.IsBlobOnly():
		before := item.Blob.String()
		after := store.envRepo.Rel(before)

		if err = object.ExternalObjectId.SetBlob(after); err != nil {
			err = errors.Wrap(err)
			return err
		}

	default:
		externalObjectId := &item.ExternalObjectId

		if err = ids.SetObjectIdOrBlob(
			&object.ObjectId,
			externalObjectId,
		); err != nil {
			if doddish.IsErrEmptySeq(err) {
				err = nil
			} else {
				err = errors.Wrapf(err, "ExternalId: %q", externalObjectId)
				return err
			}
		}

		if err = object.ExternalObjectId.SetObjectIdLike(
			&item.ExternalObjectId,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if object.ExternalObjectId.String() != externalObjectId.String() {
			err = errors.ErrorWithStackf(
				"expected %q but got %q. %s",
				externalObjectId,
				&object.ExternalObjectId,
				item.Debug(),
			)

			return err
		}
	}

	if err = item.WriteToSku(
		object,
		store.envRepo,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	fdees := quiter.SortedValues(item.FDs.All())

	for _, fdee := range fdees {
		field := objects.Field{
			Value:     fdee.GetPath(),
			ColorType: string_format_writer.ColorTypeId,
		}

		switch {
		case item.Object.Equals(fdee):
			field.Key = "object"

		case item.Conflict.Equals(fdee):
			field.Key = "conflict"

		case item.Lockfile.Equals(fdee):
			field.Key = "lockfile"

		case item.Blob.Equals(fdee):
			fallthrough

		default:
			field.Key = "blob"
		}

		object.GetMetadataMutable().GetIndexMutable().GetFieldsMutable().Append(field)
	}

	return err
}
