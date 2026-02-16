package store_fs

import (
	"fmt"
	"os"
	"path"
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter_set"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/checked_out_state"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/golf/objects"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

func (store *Store) CheckoutOne(
	options checkout_options.Options,
	sz sku.TransactedGetter,
) (col sku.SkuType, err error) {
	col, _, err = store.checkoutOneIfNecessary(options, sz)
	return col, err
}

func (store *Store) checkoutOneForReal(
	options checkout_options.Options,
	checkedOut *sku.CheckedOut,
	item *sku.FSItem,
) (err error) {
	if store.config.IsDryRun() {
		return err
	}

	fsOptions := GetCheckoutOptionsFromOptions(options)

	// delete the existing checkout if it exists in the cwd
	if fsOptions.Path == PathOptionDefault {
		if err = store.RemoveItem(item); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	var info checkoutFileNameInfo

	if err = store.hydrateCheckoutFileNameInfoFromCheckedOut(
		options,
		checkedOut,
		&info,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.setObjectIfNecessary(options, item, info); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.setLockfileIfNecessary(options, item, info); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.setBlobIfNecessary(
		options,
		item,
		info,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// This is necessary otherwise External is an empty sku
	sku.Resetter.ResetWith(checkedOut.GetSkuExternal(), checkedOut.GetSku())

	if err = store.WriteFSItemToExternal(item, checkedOut.GetSkuExternal()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.fileEncoder.Encode(
		fsOptions.TextFormatterOptions,
		checkedOut.GetSkuExternal(),
		item,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) setObjectIfNecessary(
	options checkout_options.Options,
	fsItem *sku.FSItem,
	info checkoutFileNameInfo,
) (err error) {
	if !options.CheckoutMode.IncludesMetadata() {
		quiter_set.Del(fsItem.FDs, &fsItem.Object)
		fsItem.Object.Reset()
		return err
	}

	if err = fsItem.Object.SetPath(info.objectName); err != nil {
		err = errors.Wrap(err)
		return err
	}

	fsItem.FDs.Add(&fsItem.Object)

	return err
}

func (store *Store) setLockfileIfNecessary(
	options checkout_options.Options,
	fsItem *sku.FSItem,
	info checkoutFileNameInfo,
) (err error) {
	if !options.CheckoutMode.IncludesLockfile() {
		quiter_set.Del(fsItem.FDs, &fsItem.Lockfile)
		fsItem.Lockfile.Reset()
		return err
	}

	fileExtension := store.fileExtensions.Lockfile

	if err = fsItem.Blob.SetPath(
		info.basename + "." + fileExtension,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	fsItem.FDs.Add(&fsItem.Lockfile)

	return err
}

func (store *Store) setBlobIfNecessary(
	options checkout_options.Options,
	fsItem *sku.FSItem,
	info checkoutFileNameInfo,
) (err error) {
	fsOptions := GetCheckoutOptionsFromOptions(options)

	if fsOptions.ForceInlineBlob ||
		!options.CheckoutMode.IncludesBlob() {
		quiter_set.Del(fsItem.FDs, &fsItem.Blob)
		fsItem.Blob.Reset()
		return err
	}

	fileExtension := store.config.GetTypeExtension(info.tipe.String())

	if fileExtension == "" {
		fileExtension = info.tipe.ToType().StringSansOp()
	}

	if err = fsItem.Blob.SetPath(
		info.basename + "." + fileExtension,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	fsItem.FDs.Add(&fsItem.Blob)

	return err
}

func (store *Store) shouldCheckOut(
	options checkout_options.Options,
	checkedOut *sku.CheckedOut,
	allowMutterMatch bool,
) bool {
	if options.Force {
		return true
	}

	eq := objects.EqualerSansTai.Equals(
		checkedOut.GetSku().GetMetadata(),
		checkedOut.GetSkuExternal().GetMetadata(),
	)

	if eq {
		return true
	}

	if !allowMutterMatch {
		ui.Log().Print("")
		return false
	}

	mother, motherRepool := sku.GetTransactedPool().GetWithRepool()
	defer motherRepool()

	if err := store.storeSupplies.ReadOneInto(
		checkedOut.GetSku().GetObjectId(),
		mother,
	); err == nil {
		if objects.EqualerSansTai.Equals(
			mother.GetMetadata(),
			checkedOut.GetSkuExternal().GetMetadata(),
		) {
			return true
		}
	}

	ui.Log().Print("")

	return false
}

type checkoutFileNameInfo struct {
	basename   string
	objectName string
	tipe       ids.Type
	inlineBlob bool
}

func (store *Store) hydrateCheckoutFileNameInfoFromCheckedOut(
	options checkout_options.Options,
	checkedOut *sku.CheckedOut,
	info *checkoutFileNameInfo,
) (err error) {
	if err = store.SetFilenameForTransacted(
		options,
		checkedOut.GetSku(),
		info,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	checkedOut.SetState(checked_out_state.JustCheckedOut)

	info.tipe = checkedOut.GetSku().GetType()
	info.inlineBlob = store.config.IsInlineType(info.tipe)

	return err
}

func (store *Store) SetFilenameForTransacted(
	options checkout_options.Options,
	object *sku.Transacted,
	info *checkoutFileNameInfo,
) (err error) {
	cwd := store.envRepo.GetCwd()

	fsOptions := GetCheckoutOptionsFromOptions(options)

	if fsOptions.Path == PathOptionTempLocal {
		var file *os.File

		if file, err = store.envRepo.GetTempLocal().FileTempWithTemplate(
			fmt.Sprintf(
				"*.%s",
				store.FileExtensionForObject(object),
			),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloser(&err, file)

		info.basename = file.Name()
		info.objectName = file.Name()

		return err
	}

	if object.GetGenre() == genres.Zettel {
		var zettelId ids.ZettelId

		if err = zettelId.Set(object.GetObjectId().String()); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if info.basename, err = env_dir.MakeDirIfNecessaryForStringerWithHeadAndTail(
			zettelId,
			cwd,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		info.objectName = store.PathForTransacted(cwd, object)
	} else {
		info.basename = store.PathForTransacted(cwd, object)
		info.objectName = info.basename
	}

	if strings.Contains(info.basename, "!") {
		err = errors.ErrorWithStackf(
			"contains illegal characters: %q",
			info.basename,
		)
		return err
	}

	if strings.Contains(info.objectName, "!") {
		err = errors.ErrorWithStackf(
			"contains illegal characters: %q",
			info.objectName,
		)
		return err
	}

	return err
}

func (store *Store) PathForTransacted(dir string, object *sku.Transacted) string {
	return path.Join(
		dir,
		fmt.Sprintf(
			"%s.%s",
			object.GetObjectId().StringSansOp(),
			store.FileExtensionForObject(object),
		),
	)
}

func (store *Store) FileExtensionForObject(
	object *sku.Transacted,
) string {
	var extension string

	if object.GetGenre() == genres.Blob {
		extension = store.config.GetTypeExtension(object.GetType().String())

		if extension == "" {
			extension = object.GetType().ToType().StringSansOp()
		}
	} else {
		extension = store.fileExtensions.GetFileExtensionForGenre(object)
	}

	if extension == "" {
		extension = "unknown"
	}

	return extension
}

func (store *Store) RemoveItem(fsItem *sku.FSItem) (err error) {
	// TODO check conflict state
	for fdItem := range fsItem.FDs.All() {
		if err = fdItem.Remove(store.envRepo); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	fsItem.Reset()

	return err
}

func (store *Store) UpdateCheckoutFromCheckedOut(
	options checkout_options.OptionsWithoutMode,
	object sku.SkuType,
) (err error) {
	checkoutOptions := checkout_options.Options{
		OptionsWithoutMode: options,
	}

	if checkoutOptions.CheckoutMode, err = store.GetCheckoutMode(
		object.GetSkuExternal(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if checkoutOptions.CheckoutMode.IsEmpty() {
		return err
	}

	fsOptions := GetCheckoutOptionsFromOptionsWithoutMode(options)
	fsOptions.Path = PathOptionTempLocal
	options.StoreSpecificOptions = fsOptions

	var oldFDs, newFDs *sku.FSItem

	if oldFDs, err = store.ReadFSItemFromExternal(object.GetSkuExternal()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if _, newFDs, err = store.checkoutOneIfNecessary(
		checkoutOptions,
		object.GetSkuExternal(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if !oldFDs.Object.IsEmpty() &&
		!newFDs.Object.IsEmpty() &&
		!store.config.IsDryRun() {
		if err = os.Rename(
			newFDs.Object.GetPath(),
			oldFDs.Object.GetPath(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if !oldFDs.Blob.IsEmpty() &&
		!newFDs.Blob.IsEmpty() &&
		!store.config.IsDryRun() {
		if err = os.Rename(
			newFDs.Blob.GetPath(),
			oldFDs.Blob.GetPath(),
		); err != nil {
			err = errors.Wrapf(
				err,
				"New: %q, Old: %q",
				newFDs.Blob.GetPath(),
				oldFDs.Blob.GetPath(),
			)

			return err
		}
	}

	return err
}
