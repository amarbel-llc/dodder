package box_format

import (
	"code.linenisgreat.com/dodder/go/internal/delta/genres"
	"code.linenisgreat.com/dodder/go/internal/delta/options_print"
	"code.linenisgreat.com/dodder/go/internal/echo/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/fd"
	"code.linenisgreat.com/dodder/go/internal/india/env_dir"
	"code.linenisgreat.com/dodder/go/internal/india/object_metadata_box_builder"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
)

func MakeBoxCheckedOut(
	co string_format_writer.ColorOptions,
	options options_print.Options,
	boxWriter interfaces.StringEncoderTo[string_format_writer.Box],
	abbr ids.Abbr,
	fsItemReadWriter sku.FSItemReadWriter,
	relativePath env_dir.RelativePath,
	headerWriter string_format_writer.HeaderWriter[*sku.CheckedOut],
) *BoxCheckedOut {
	return &BoxCheckedOut{
		headerWriter: headerWriter,
		BoxTransacted: BoxTransacted{
			optionsColor:     co,
			optionsPrint:     options,
			boxStringEncoder: boxWriter,
			abbr:             abbr,
			fsItemReadWriter: fsItemReadWriter,
			relativePath:     relativePath,
		},
	}
}

type BoxCheckedOut struct {
	headerWriter string_format_writer.HeaderWriter[*sku.CheckedOut]
	BoxTransacted
}

func (format *BoxCheckedOut) EncodeStringTo(
	co *sku.CheckedOut,
	sw interfaces.WriterAndStringWriter,
) (n int64, err error) {
	// TODO pool boxes
	var box string_format_writer.Box
	builder := (*object_metadata_box_builder.Builder)(&box)

	if format.headerWriter != nil {
		if err = format.headerWriter.WriteBoxHeader(&box.Header, co); err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	box.Contents.Grow(10)

	var fds *sku.FSItem
	var errFS error

	external := co.GetSkuExternal()

	if format.fsItemReadWriter != nil {
		fds, errFS = format.fsItemReadWriter.ReadFSItemFromExternal(external)
	}

	if format.fsItemReadWriter == nil || errFS != nil ||
		!external.RepoId.IsEmpty() {
		if err = format.addFieldsExternalWithFSItem(
			external,
			builder,
			format.optionsPrint.BoxDescriptionInBox,
			fds,
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	} else {
		if err = format.addFieldsFS(co, builder, fds); err != nil {
			err = errors.Wrapf(err, "CheckedOut: %s", co)
			return n, err
		}
	}

	b := external.GetMetadata().GetDescription()

	if !format.optionsPrint.BoxDescriptionInBox && !b.IsEmpty() {
		box.Trailer.Append(string_format_writer.Field{
			Value:              b.StringWithoutNewlines(),
			ColorType:          string_format_writer.ColorTypeUserData,
			DisableValueQuotes: true,
		})
	}

	if n, err = format.boxStringEncoder.EncodeStringTo(box, sw); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (format *BoxCheckedOut) addFieldsExternalWithFSItem(
	external *sku.Transacted,
	builder *object_metadata_box_builder.Builder,
	includeDescriptionInBox bool,
	item *sku.FSItem,
) (err error) {
	if err = format.addFieldsObjectIdsWithFSItem(
		external,
		builder,
		true,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = format.addFieldsMetadataWithFSItem(
		format.optionsPrint,
		external,
		includeDescriptionInBox,
		builder,
		item,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// func (f *BoxTransacted) makeFieldExternalObjectIdsIfNecessary(
// 	sk *sku.Transacted,
// 	item *sku.FSItem,
// ) (field string_format_writer.Field, err error) {
// 	field = string_format_writer.Field{
// 		ColorType: string_format_writer.ColorTypeId,
// 	}

// 	switch {
// 	case (item == nil || item.Len() == 0) && !sk.ExternalObjectId.IsEmpty():
// 		oid := &sk.ExternalObjectId
// 		// TODO quote as necessary
// 		field.Value = (&ids.ObjectIdStringerSansRepo{oid}).String()

// 	case item != nil:
// 		switch item.GetCheckoutMode() {
// 		case checkout_mode.MetadataOnly, checkout_mode.MetadataAndBlob:
// 			// TODO quote as necessary
// 			if !item.Object.IsStdin() {
// 				field.Value = f.relativePath.Rel(item.Object.String())
// 			}
// 		}
// 	}

// 	return
// }

func (format *BoxCheckedOut) makeFieldObjectId(
	object *sku.Transacted,
) (field string_format_writer.Field, empty bool, err error) {
	objectId := &object.ObjectId

	empty = objectId.IsEmpty()

	oidString := (&ids.StringerSansRepo{Id: objectId}).String()

	if format.abbr.ZettelId.Abbreviate != nil &&
		objectId.GetGenre() == genres.Zettel {
		if oidString, err = format.abbr.ZettelId.Abbreviate(objectId); err != nil {
			err = errors.Wrap(err)
			return field, empty, err
		}
	}

	field = string_format_writer.Field{
		Value:     oidString,
		ColorType: string_format_writer.ColorTypeId,
	}

	return field, empty, err
}

func (format *BoxTransacted) addFieldsObjectIdsWithFSItem(
	object *sku.Transacted,
	builder *object_metadata_box_builder.Builder,
	preferExternal bool,
) (err error) {
	var external string_format_writer.Field

	if external, err = format.makeFieldExternalObjectIdsIfNecessary(
		object,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var internal string_format_writer.Field
	var internalEmpty bool

	if internal, internalEmpty, err = format.makeFieldObjectId(object); err != nil {
		err = errors.Wrap(err)
		return err
	}

	switch {
	case (internalEmpty || preferExternal) && external.Value != "":
		builder.Contents.Append(external)

	case internal.Value != "":
		builder.Contents.Append(internal)

	case external.Value != "":
		builder.Contents.Append(external)

	default:
		err = errors.Errorf("empty id")
		return err
	}

	return err
}

func (format *BoxCheckedOut) addFieldsMetadataWithFSItem(
	options options_print.Options,
	object *sku.Transacted,
	includeDescriptionInBox bool,
	builder *object_metadata_box_builder.Builder,
	item *sku.FSItem,
) (err error) {
	metadata := object.GetMetadataMutable()

	if options.PrintBlobDigests &&
		(options.BoxPrintEmptyBlobIds || !metadata.GetBlobDigest().IsNull()) {
		builder.AddBlobDigestIfNecessary(
			metadata.GetBlobDigest(),
			format.abbr.BlobId.Abbreviate,
		)
	}

	if options.BoxPrintTai && object.GetGenre() != genres.InventoryList {
		builder.AddTai(metadata)
	}

	if !metadata.GetType().IsEmpty() {
		builder.AddType(metadata)
	}

	b := metadata.GetDescription()

	if includeDescriptionInBox && !b.IsEmpty() {
		builder.AddDescription(metadata)
	}

	builder.AddTags(metadata)

	if !options.BoxExcludeFields &&
		(item == nil || item.FDs.Len() == 0) {
		quiter.AppendSeq(&builder.Contents, metadata.GetIndex().GetFields())
	}

	return err
}

func (format *BoxCheckedOut) addFieldsFS(
	checkedOut *sku.CheckedOut,
	builder *object_metadata_box_builder.Builder,
	item *sku.FSItem,
) (err error) {
	mode := item.GetCheckoutMode()

	var fdAlreadyWritten *fd.FD

	op := format.optionsPrint
	op.BoxExcludeFields = true

	switch checkedOut.GetState() {
	case checked_out_state.Unknown:
		err = errors.ErrorWithStackf("invalid state unknown")
		return err

	case checked_out_state.Untracked:
		if err = format.addFieldsUntracked(checkedOut, builder, item, op); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err

	case checked_out_state.Recognized:
		if err = format.addFieldsRecognized(checkedOut, builder, item, op); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}

	var id string_format_writer.Field

	switch {
	case mode.IsBlobOnly() || mode.IsBlobRecognized():
		id.Value = (&ids.StringerSansRepo{Id: &checkedOut.GetSkuExternal().ObjectId}).String()

	case mode.IncludesMetadata():
		id.Value = format.relativePath.Rel(item.Object.GetPath())
		fdAlreadyWritten = &item.Object

	default:
		if id, _, err = format.makeFieldObjectId(
			checkedOut.GetSkuExternal(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	id.ColorType = string_format_writer.ColorTypeId
	builder.Contents.Append(id)

	if checkedOut.GetState() == checked_out_state.Conflicted {
		// TODO handle conflicted state
	} else {
		if err = format.addFieldsMetadata(
			op,
			checkedOut.GetSkuExternal(),
			op.BoxDescriptionInBox,
			builder,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if mode.IsBlobRecognized() ||
			(!mode.IsMetadataOnly() && !mode.IsEmpty()) {
			if err = format.addFieldsFSBlobExcept(
				item,
				fdAlreadyWritten,
				builder,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}
	}

	return err
}

func (format *BoxTransacted) addFieldsFSBlobExcept(
	fds *sku.FSItem,
	except *fd.FD,
	builder *object_metadata_box_builder.Builder,
) (err error) {
	if fds.FDs == nil {
		err = errors.ErrorWithStackf("FDSet.MutableSetLike was nil")
		return err
	}

	for fd := range fds.FDs.All() {
		if except != nil && fd.Equals(except) {
			continue
		}

		if err = format.addFieldFSBlob(fd, builder); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (format *BoxTransacted) addFieldFSBlob(
	fd *fd.FD,
	builder *object_metadata_box_builder.Builder,
) (err error) {
	builder.Contents.Append(
		string_format_writer.Field{
			Value:        format.relativePath.Rel(fd.GetPath()),
			ColorType:    string_format_writer.ColorTypeId,
			NeedsNewline: true,
		},
	)

	return err
}

func (format *BoxTransacted) addFieldsUntracked(
	checkedOut *sku.CheckedOut,
	builder *object_metadata_box_builder.Builder,
	item *sku.FSItem,
	op options_print.Options,
) (err error) {
	// fdToPrint := &item.Blob

	// if co.GetSkuExternal().GetGenre() != genres.Zettel &&
	// !item.Object.IsEmpty() {
	// 	fdToPrint = &item.Object
	// }

	// box.Contents = append(
	// 	box.Contents,
	// 	string_format_writer.Field{
	// 		ColorType: string_format_writer.ColorTypeId,
	// 		Value:     f.relativePath.Rel(fdToPrint.GetPath()),
	// 	},
	// )
	if err = format.addFieldsObjectIdsWithFSItem(
		checkedOut.GetSkuExternal(),
		builder,
		true,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = format.addFieldsMetadata(
		op,
		checkedOut.GetSkuExternal(),
		format.optionsPrint.BoxDescriptionInBox,
		builder,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (format *BoxTransacted) addFieldsRecognized(
	co *sku.CheckedOut,
	builder *object_metadata_box_builder.Builder,
	item *sku.FSItem,
	op options_print.Options,
) (err error) {
	var id string_format_writer.Field

	if id, _, err = format.makeFieldObjectId(
		co.GetSkuExternal(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	id.ColorType = string_format_writer.ColorTypeId
	builder.Contents.Append(id)

	if err = format.addFieldsMetadata(
		op,
		co.GetSkuExternal(),
		op.BoxDescriptionInBox,
		builder,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = format.addFieldsFSBlobExcept(
		item,
		nil,
		builder,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
