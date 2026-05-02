package object_metadata_fmt

import (
	"sort"

	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

func MetadataFieldError(
	err error,
) []string_format_writer.FormattedField {
	var errorGroup errors.Group

	if errors.As(err, &errorGroup) {
		out := make([]string_format_writer.FormattedField, 0, errorGroup.Len())

		for _, e := range errorGroup {
			out = append(
				out,
				string_format_writer.FormattedField{
					Field: fields.Field{
						Key:   "error",
						Value: e.Error(),
						Type:  fields.TypeUserData,
					},
					NoTruncate: true,
				},
			)
		}

		return out
	} else {
		return []string_format_writer.FormattedField{
			{
				Field: fields.Field{
					Key:   "error",
					Value: err.Error(),
					Type:  fields.TypeUserData,
				},
				NoTruncate: true,
			},
		}
	}
}

func MetadataFieldTai(
	metadata objects.MetadataMutable,
) string_format_writer.FormattedField {
	return string_format_writer.FormattedField{
		Field: fields.Field{
			Value: metadata.GetTai().String(),
			Type:  fields.TypeHash,
		},
	}
}

func MetadataFieldType(
	metadata objects.MetadataMutable,
) string_format_writer.FormattedField {
	return string_format_writer.FormattedField{
		Field: fields.Field{
			Value: metadata.GetType().String(),
			Type:  fields.TypeType,
		},
	}
}

func MetadataFieldTags(
	metadata objects.MetadataMutable,
) []string_format_writer.FormattedField {
	tags := make([]string_format_writer.FormattedField, 0, metadata.GetTags().Len())

	for t := range metadata.AllTags() {
		tags = append(
			tags,
			string_format_writer.FormattedField{
				Field: fields.Field{
					Value: t.String(),
				},
			},
		)
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Value < tags[j].Value
	})

	return tags
}

func MetadataFieldDescription(
	metadata objects.MetadataMutable,
) string_format_writer.FormattedField {
	return string_format_writer.FormattedField{
		Field: fields.Field{
			Value: metadata.GetDescription().StringWithoutNewlines(),
			Type:  fields.TypeUserData,
		},
	}
}
