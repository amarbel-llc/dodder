package box_format

import (
	"fmt"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/_/options_print"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/env_ui"
	"code.linenisgreat.com/dodder/go/internal/echo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/echo/object_metadata_box_builder"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
)

func MakeBoxTransactedArchive(
	env env_ui.Env,
	optionsOriginal options_print.Options,
) *BoxTransacted {
	options := optionsOriginal.
		WithPrintBlobDigests(true).
		WithExcludeFields(true).
		WithDescriptionInBox(true)

	colorOptions := env.FormatColorOptionsOut(optionsOriginal)
	colorOptions.OffEntirely = true

	format := MakeBoxTransacted(
		colorOptions,
		options,
		env.StringFormatWriterFields(
			string_format_writer.CliFormatTruncationNone,
			colorOptions,
		),
		ids.Abbr{},
		nil,
		nil,
		nil,
	)

	format.isArchive = true

	return format
}

func MakeBoxTransacted(
	colorOptions string_format_writer.ColorOptions,
	options options_print.Options,
	boxStringEncoder interfaces.StringEncoderTo[string_format_writer.Box],
	abbr ids.Abbr,
	fsItemReadWriter sku.FSItemReadWriter,
	relativePath env_dir.RelativePath,
	headerWriter string_format_writer.HeaderWriter[*sku.Transacted],
) *BoxTransacted {
	return &BoxTransacted{
		optionsColor:     colorOptions,
		optionsPrint:     options,
		boxStringEncoder: boxStringEncoder,
		abbr:             abbr,
		fsItemReadWriter: fsItemReadWriter,
		relativePath:     relativePath,
		headerWriter:     headerWriter,
	}
}

type BoxTransacted struct {
	optionsColor string_format_writer.ColorOptions
	optionsPrint options_print.Options

	boxStringEncoder interfaces.StringEncoderTo[string_format_writer.Box]
	headerWriter     string_format_writer.HeaderWriter[*sku.Transacted]

	abbr             ids.Abbr
	fsItemReadWriter sku.FSItemReadWriter
	relativePath     env_dir.RelativePath

	isArchive bool
}

func (format *BoxTransacted) EncodeStringTo(
	object *sku.Transacted,
	writer interfaces.WriterAndStringWriter,
) (n int64, err error) {
	var box string_format_writer.Box
	builder := (*object_metadata_box_builder.Builder)(&box)

	// box.Header.RightAligned = true

	// if f.optionsPrint.PrintTime && !f.optionsPrint.PrintTai {
	// 	t := sk.GetTai()
	// 	box.Header.Value = t.Format(string_format_writer.StringFormatDateTime)
	// }

	if format.headerWriter != nil {
		if err = format.headerWriter.WriteBoxHeader(&box.Header, object); err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	box.Contents = slices.Grow(box.Contents, 10)

	if err = format.addFieldsObjectIds(
		object,
		builder,
	); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if err = format.addFieldsMetadata(
		format.optionsPrint,
		object,
		format.optionsPrint.BoxDescriptionInBox,
		builder,
	); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	b := object.GetMetadata().GetDescription()

	if !format.optionsPrint.BoxDescriptionInBox && !b.IsEmpty() {
		box.Trailer = append(
			box.Trailer,
			string_format_writer.Field{
				Value:              b.StringWithoutNewlines(),
				ColorType:          string_format_writer.ColorTypeUserData,
				DisableValueQuotes: true,
			},
		)
	}

	if n, err = format.boxStringEncoder.EncodeStringTo(box, writer); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (format *BoxTransacted) makeFieldExternalObjectIdsIfNecessary(
	object *sku.Transacted,
) (field string_format_writer.Field, err error) {
	field = string_format_writer.Field{
		ColorType: string_format_writer.ColorTypeId,
	}

	if !object.ExternalObjectId.IsEmpty() {
		objectId := &object.ExternalObjectId
		// TODO quote as necessary
		field.Value = (&ids.StringerSansRepo{Id: objectId}).String()
	}

	return field, err
}

func (format *BoxTransacted) makeFieldObjectId(
	object *sku.Transacted,
) (field string_format_writer.Field, empty bool, err error) {
	objectId := &object.ObjectId

	empty = objectId.IsEmpty()

	objectIdString := (&ids.StringerSansRepo{Id: objectId}).String()

	if format.abbr.ZettelId.Abbreviate != nil &&
		objectId.GetGenre() == genres.Zettel &&
		!empty {

		if objectIdString, err = format.abbr.ZettelId.Abbreviate(objectId); err != nil {
			err = errors.Wrap(err)
			return field, empty, err
		}
	}

	field = string_format_writer.Field{
		Value:              objectIdString,
		DisableValueQuotes: true,
		ColorType:          string_format_writer.ColorTypeId,
	}

	return field, empty, err
}

func (format *BoxTransacted) addFieldsObjectIds(
	object *sku.Transacted,
	builder *object_metadata_box_builder.Builder,
) (err error) {
	var external string_format_writer.Field

	if external, err = format.makeFieldExternalObjectIdsIfNecessary(
		object,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var internal string_format_writer.Field
	var externalEmpty bool

	if internal, externalEmpty, err = format.makeFieldObjectId(object); err != nil {
		err = errors.Wrap(err)
		return err
	}

	switch {
	// case internal.Value != "" && external.Value != "":
	// 	if strings.HasPrefix(external.Value, strings.TrimPrefix(internal.Value,
	// "!")) {
	// 		box.Contents = append(box.Contents, external)
	// 	} else {
	// 		box.Contents = append(box.Contents, internal, external)
	// 	}

	case externalEmpty && external.Value != "":
		builder.Contents.Append(external)

	case internal.Value != "":
		builder.Contents.Append(internal)

	case external.Value != "":
		builder.Contents.Append(external)

	default:
		panic(fmt.Sprintf("empty object id: %q", sku.String(object)))
	}

	return err
}

func (format *BoxTransacted) addFieldsMetadata(
	options options_print.Options,
	object *sku.Transacted,
	includeDescriptionInBox bool,
	builder *object_metadata_box_builder.Builder,
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

	if format.isArchive && !object.GetMetadata().GetObjectSig().IsNull() {
		builder.AddRepoPubKey(metadata)
		builder.AddMotherSigIfNecessary(metadata)
		builder.AddObjectSig(metadata)
	}

	if format.isArchive {
		builder.AddTypeAndLock(metadata)
	} else if !metadata.GetType().IsEmpty() {
		builder.AddType(metadata)
	}

	description := metadata.GetDescription()

	if includeDescriptionInBox && !description.IsEmpty() {
		builder.AddDescription(metadata)
	}

	builder.AddTags(metadata)

	if !options.BoxExcludeFields && !format.isArchive {
		quiter.AppendSeq(&builder.Contents, metadata.GetIndex().GetFields())
	}

	return err
}
