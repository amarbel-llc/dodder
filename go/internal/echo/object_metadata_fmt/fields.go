package object_metadata_fmt

import (
	"sort"

	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func MetadataFieldError(
	err error,
) []string_format_writer.Field {
	var errorGroup errors.Group

	if errors.As(err, &errorGroup) {
		out := make([]string_format_writer.Field, 0, errorGroup.Len())

		for _, e := range errorGroup {
			out = append(
				out,
				string_format_writer.Field{
					Key:        "error",
					Value:      e.Error(),
					ColorType:  string_format_writer.ColorTypeUserData,
					NoTruncate: true,
				},
			)
		}

		return out
	} else {
		return []string_format_writer.Field{
			{
				Key:        "error",
				Value:      err.Error(),
				ColorType:  string_format_writer.ColorTypeUserData,
				NoTruncate: true,
			},
		}
	}
}

func MetadataFieldTai(
	metadata objects.MetadataMutable,
) string_format_writer.Field {
	return string_format_writer.Field{
		Value:     metadata.GetTai().String(),
		ColorType: string_format_writer.ColorTypeHash,
	}
}

func MetadataFieldType(
	metadata objects.MetadataMutable,
) string_format_writer.Field {
	return string_format_writer.Field{
		Value:     metadata.GetType().String(),
		ColorType: string_format_writer.ColorTypeType,
	}
}

func MetadataFieldTags(
	metadata objects.MetadataMutable,
) []string_format_writer.Field {
	tags := make([]string_format_writer.Field, 0, metadata.GetTags().Len())

	for t := range metadata.AllTags() {
		tags = append(
			tags,
			string_format_writer.Field{
				Value: t.String(),
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
) string_format_writer.Field {
	return string_format_writer.Field{
		Value:     metadata.GetDescription().StringWithoutNewlines(),
		ColorType: string_format_writer.ColorTypeUserData,
	}
}
