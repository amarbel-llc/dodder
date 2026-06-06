package object_metadata_fmt

import (
	"fmt"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/lib/0/collections_slice"
)

func AddBlobDigestIfNecessary(
	boxContents collections_slice.Slice[string_format_writer.FormattedField],
	digest mad_domain_interfaces.MarklId,
	funcAbbreviate domain_interfaces.FuncAbbreviateString,
) {
	value := digest.String()

	if funcAbbreviate != nil {
		abbreviatedDigestString, err := funcAbbreviate(digest)
		if err != nil {
			panic(err)
		}

		// An empty result means the abbreviation index has no entries to
		// abbreviate against (e.g. during clone/genesis, before the index
		// is populated). Fall back to the full digest.
		if abbreviatedDigestString != "" {
			value = abbreviatedDigestString
		}
	}

	if value == "" {
		return
	}

	field := string_format_writer.FormattedField{
		Field: fields.Field{
			Value: "@" + value,
			Type:  fields.TypeHash,
		},
		NoTruncate: true,
	}

	boxContents.Append(field)
}

func AddRepoPubKey(
	boxContents collections_slice.Slice[string_format_writer.FormattedField],
	metadata objects.MetadataMutable,
) {
	addMarklIdIfNotNull(
		boxContents,
		metadata.GetRepoPubKey(),
	)
}

func AddObjectSig(
	boxContents collections_slice.Slice[string_format_writer.FormattedField],
	metadata objects.MetadataMutable,
) {
	boxContents.Append(
		makeMarklIdField(metadata.GetObjectSig()),
	)
}

func AddMotherSigIfNecessary(
	boxContents collections_slice.Slice[string_format_writer.FormattedField],
	metadata objects.MetadataMutable,
) {
	addMarklIdIfNotNull(
		boxContents,
		metadata.GetMotherObjectSig(),
	)
}

func AddReferencedObject(
	boxContents collections_slice.Slice[string_format_writer.FormattedField],
	metadata objects.MetadataMutable,
) {
	boxContents.Append(
		makeMarklIdField(metadata.GetObjectSig()),
	)
}

func addMarklIdIfNotNull(
	boxContents collections_slice.Slice[string_format_writer.FormattedField],
	id mad_domain_interfaces.MarklId,
) {
	if id.IsNull() {
		return
	}

	addMarklId(boxContents, id)
}

func addMarklId(
	boxContents collections_slice.Slice[string_format_writer.FormattedField],
	id mad_domain_interfaces.MarklId,
) {
	boxContents.Append(
		makeMarklIdField(id),
	)
}

func makeMarklIdField(
	id mad_domain_interfaces.MarklId,
) string_format_writer.FormattedField {
	if id.GetPurposeId() == "" {
		panic(fmt.Sprintf("empty format for markl id: %q", id))
	}

	return string_format_writer.FormattedField{
		Field: fields.Field{
			Key:   id.GetPurposeId(),
			Value: id.String(),
			Type:  fields.TypeHash,
		},
		Separator:  '@',
		NoTruncate: true,
	}
}
