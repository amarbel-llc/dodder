package box_format

import (
	"fmt"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/echo/object_metadata_box_builder"
	"code.linenisgreat.com/dodder/go/internal/echo/repo_identity"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	mad_env_dir "github.com/amarbel-llc/madder/go/pkgs/env_dir"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func MakeBoxTransactedArchive(
	env env_ui.Env,
	optionsOriginal options_print.Options,
) *BoxTransacted {
	options := optionsOriginal.
		WithPrintBlobDigests(true).
		WithExcludeFields(true).
		WithDescriptionInBox(true).
		WithPrintSigs(true)

	colorOptions := env_ui.FormatColorOptionsOut(env, optionsOriginal)
	colorOptions.OffEntirely = true

	format := MakeBoxTransacted(
		colorOptions,
		options,
		env_ui.StringFormatWriterFields(string_format_writer.CliFormatTruncationNone, colorOptions),
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
	relativePath mad_env_dir.RelativePath,
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
	relativePath     mad_env_dir.RelativePath

	// selfPubKey + selfHandle carry the local repo's provenance identity.
	// When an object's GetRepoPubKey() matches selfPubKey, the pubkey renders
	// as the `<handle>@<pubkey>` self form under -print-sigs (see
	// addFieldsMetadata). Both zero by default; an unset selfPubKey degrades to
	// today's bare pubkey rendering. Display-only: only the user-facing
	// printers (romeo/local_working_copy) set these --- the inventory-list wire
	// coder and other internal / archive constructors leave them unset so
	// persisted / exported bytes stay bare.
	selfPubKey markl.Id
	selfHandle string

	isArchive bool
}

func (format *BoxTransacted) SetAbbr(abbr ids.Abbr) {
	format.abbr = abbr
}

// SetSelfProvenance stamps the local repo's identity onto the formatter so
// objects authored by this repo render their provenance as
// `<handle>@<pubkey>` under -print-sigs, distinguishing them from foreign
// provenance (bare pubkey). A null pubkey leaves self-provenance unset.
func (format *BoxTransacted) SetSelfProvenance(
	pubKey mad_domain_interfaces.MarklId,
	handle string,
) {
	if pubKey != nil && !pubKey.IsNull() {
		format.selfPubKey.ResetWithMarklId(pubKey)
	}

	format.selfHandle = handle
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
			string_format_writer.FormattedField{
				Field: fields.Field{
					Value: b.StringWithoutNewlines(),
					Type:  fields.TypeUserData,
				},
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
) (field string_format_writer.FormattedField, err error) {
	field = string_format_writer.FormattedField{
		Field: fields.Field{
			Type: fields.TypeId,
		},
	}

	if !object.GetExternalObjectId().IsEmpty() {
		objectId := object.GetExternalObjectIdMutable()
		// TODO quote as necessary
		field.Value = (&ids.StringerSansRepo{Id: objectId}).String()
	}

	return field, err
}

func (format *BoxTransacted) makeFieldObjectId(
	object *sku.Transacted,
) (field string_format_writer.FormattedField, empty bool, err error) {
	objectId := object.GetObjectId()

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

	field = string_format_writer.FormattedField{
		Field: fields.Field{
			Value: objectIdString,
			Type:  fields.TypeId,
		},
		DisableValueQuotes: true,
	}

	return field, empty, err
}

func (format *BoxTransacted) addFieldsObjectIds(
	object *sku.Transacted,
	builder *object_metadata_box_builder.Builder,
) (err error) {
	var external string_format_writer.FormattedField

	if external, err = format.makeFieldExternalObjectIdsIfNecessary(
		object,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var internal string_format_writer.FormattedField
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

	if options.PrintSigs && !object.GetMetadata().GetObjectSig().IsNull() {
		objectPubKey := metadata.GetRepoPubKey()

		if !objectPubKey.IsNull() &&
			!format.selfPubKey.IsNull() &&
			markl.Equals(objectPubKey, format.selfPubKey) {
			// self provenance: object authored by this repo
			builder.AddRepoIdentity(
				repo_identity.Render(format.selfHandle, objectPubKey),
			)
		} else {
			// foreign / legacy provenance: bare (abbreviated) pubkey
			builder.AddRepoPubKey(metadata, format.abbr.PubKey.Abbreviate)
		}

		builder.AddMotherSigIfNecessary(metadata, format.abbr.Sig.Abbreviate)
		builder.AddObjectSig(metadata, format.abbr.Sig.Abbreviate)
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
	builder.AddReferencedObjectsAndLocks(metadata)
	builder.AddBlobReferences(metadata)

	if !options.BoxExcludeFields && !format.isArchive {
		for field := range metadata.GetIndex().GetFields() {
			builder.Contents.Append(string_format_writer.FormattedField{Field: field})
		}
	}

	return err
}
