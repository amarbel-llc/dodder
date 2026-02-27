package object_metadata_box_builder

import (
	"fmt"
	"sort"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/echo/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/hotel/objects"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

type Builder string_format_writer.Box

func (builder *Builder) AddBlobDigestIfNecessary(
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

	builder.Contents.Append(field)
}

func (builder *Builder) AddRepoPubKey(
	metadata objects.MetadataMutable,
) {
	builder.addMarklIdIfNotNull(metadata.GetRepoPubKey())
}

func (builder *Builder) AddObjectSig(metadata objects.MetadataMutable) {
	builder.addMarklId(metadata.GetObjectSig())
}

func (builder *Builder) AddMotherSigIfNecessary(
	metadata objects.MetadataMutable,
) {
	builder.addMarklIdIfNotNull(metadata.GetMotherObjectSig())
}

func (builder *Builder) addMarklIdIfNotNull(id domain_interfaces.MarklId) {
	if id.IsNull() {
		return
	}

	builder.addMarklId(id)
}

func (builder *Builder) addMarklId(id domain_interfaces.MarklId) {
	builder.addMarklIdWithColorType(id, id.GetPurposeId(), string_format_writer.ColorTypeHash)
}

func (builder *Builder) addMarklIdLockWithColorType(
	key string,
	value domain_interfaces.MarklId,
	colorType string_format_writer.ColorType,
) {
	builder.addMarklIdWithColorType(value, key, colorType)
}

func (builder *Builder) addMarklIdWithColorType(
	value domain_interfaces.MarklId,
	key string,
	colorType string_format_writer.ColorType,
) {
	if key == "" {
		panic(fmt.Sprintf("empty key for markl id: %q", value))
	}

	builder.Contents.Append(string_format_writer.Field{
		Key:        key,
		Separator:  '@',
		Value:      value.String(),
		NoTruncate: true,
		ColorType:  colorType,
	})
}

func (builder *Builder) AddError(err error) {
	var errorGroup errors.Group

	if errors.As(err, &errorGroup) {
		for _, err := range errorGroup {
			builder.Contents.Append(
				string_format_writer.Field{
					Key:        "error",
					Value:      err.Error(),
					ColorType:  string_format_writer.ColorTypeUserData,
					NoTruncate: true,
				},
			)
		}
	} else {
		builder.Contents.Append(
			string_format_writer.Field{
				Key:        "error",
				Value:      err.Error(),
				ColorType:  string_format_writer.ColorTypeUserData,
				NoTruncate: true,
			},
		)
	}
}

func (builder *Builder) AddTai(metadata objects.MetadataMutable) {
	builder.Contents.Append(string_format_writer.Field{
		Value:     metadata.GetTai().String(),
		ColorType: string_format_writer.ColorTypeHash,
	})
}

func (builder *Builder) AddType(
	metadata objects.MetadataMutable,
) {
	builder.Contents.Append(string_format_writer.Field{
		Value:     metadata.GetType().String(),
		ColorType: string_format_writer.ColorTypeType,
	})
}

func (builder *Builder) AddTypeAndLock(
	metadata objects.MetadataMutable,
) {
	typeLock := metadata.GetTypeLock()

	if typeLock.GetValue().IsEmpty() {
		builder.AddType(metadata)
	} else {
		builder.addMarklIdLockWithColorType(
			typeLock.GetKey().String(),
			typeLock.GetValue(),
			string_format_writer.ColorTypeType,
		)
	}
}

func (builder *Builder) AddTags(metadata objects.MetadataMutable) {
	tagCount := metadata.GetTags().Len()
	builder.Contents.Grow(tagCount)

	for tag := range metadata.AllTags() {
		builder.Contents.Append(string_format_writer.Field{
			Value: tag.String(),
		})
	}

	tags := builder.Contents[builder.Contents.Len()-tagCount:]

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Value < tags[j].Value
	})
}

func (builder *Builder) AddTagsAndLocks(metadata objects.MetadataMutable) {
	panic(errors.Err405MethodNotAllowed)
}

func (builder *Builder) AddDescription(metadata objects.MetadataMutable) {
	builder.Contents.Append(string_format_writer.Field{
		Value:     metadata.GetDescription().StringWithoutNewlines(),
		ColorType: string_format_writer.ColorTypeUserData,
	})
}
