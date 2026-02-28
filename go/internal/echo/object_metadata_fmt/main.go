package object_metadata_fmt

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func AddBlobDigestIfNecessary(
	boxContents collections_slice.Slice[string_format_writer.Field],
	digest domain_interfaces.MarklId,
	funcAbbreviate domain_interfaces.FuncAbbreviateString,
) {
	value := digest.String()

	if funcAbbreviate != nil {
		abbreviatedDigestString, err := funcAbbreviate(digest)
		if err != nil {
			panic(err)
		}

		if abbreviatedDigestString != "" {
			value = abbreviatedDigestString
		} else {
			ui.Todo("abbreviation func produced empty string")
		}
	}

	if value == "" {
		return
	}

	field := string_format_writer.Field{
		Value:      "@" + value,
		ColorType:  string_format_writer.ColorTypeHash,
		NoTruncate: true,
	}

	boxContents.Append(field)
}

func AddRepoPubKey(
	boxContents collections_slice.Slice[string_format_writer.Field],
	metadata objects.MetadataMutable,
) {
	addMarklIdIfNotNull(
		boxContents,
		metadata.GetRepoPubKey(),
	)
}

func AddObjectSig(
	boxContents collections_slice.Slice[string_format_writer.Field],
	metadata objects.MetadataMutable,
) {
	boxContents.Append(
		makeMarklIdField(metadata.GetObjectSig()),
	)
}

func AddMotherSigIfNecessary(
	boxContents collections_slice.Slice[string_format_writer.Field],
	metadata objects.MetadataMutable,
) {
	addMarklIdIfNotNull(
		boxContents,
		metadata.GetMotherObjectSig(),
	)
}

func AddReferencedObject(
	boxContents collections_slice.Slice[string_format_writer.Field],
	metadata objects.MetadataMutable,
) {
	boxContents.Append(
		makeMarklIdField(metadata.GetObjectSig()),
	)
}

func addMarklIdIfNotNull(
	boxContents collections_slice.Slice[string_format_writer.Field],
	id domain_interfaces.MarklId,
) {
	if id.IsNull() {
		return
	}

	addMarklId(boxContents, id)
}

func addMarklId(
	boxContents collections_slice.Slice[string_format_writer.Field],
	id domain_interfaces.MarklId,
) {
	boxContents.Append(
		makeMarklIdField(id),
	)
}

func makeMarklIdField(
	id domain_interfaces.MarklId,
) string_format_writer.Field {
	if id.GetPurposeId() == "" {
		panic(fmt.Sprintf("empty format for markl id: %q", id))
	}

	return string_format_writer.Field{
		Key:        id.GetPurposeId(),
		Separator:  '@',
		Value:      id.String(),
		NoTruncate: true,
		ColorType:  string_format_writer.ColorTypeHash,
	}
}
