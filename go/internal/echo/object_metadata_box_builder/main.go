package object_metadata_box_builder

import (
	"fmt"
	"sort"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

type Builder string_format_writer.Box

func (builder *Builder) AddBlobDigestIfNecessary(
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

	builder.Contents.Append(field)
}

func (builder *Builder) AddRepoPubKey(
	metadata objects.MetadataMutable,
	funcAbbreviate domain_interfaces.FuncAbbreviateString,
) {
	id := metadata.GetRepoPubKey()

	if id.IsNull() {
		return
	}

	if funcAbbreviate == nil {
		builder.addMarklId(id)
	} else {
		builder.addMarklIdAbbreviated(id, funcAbbreviate)
	}
}

// AddRepoIdentity appends a pre-rendered repo-identity provenance field (the
// self `<handle>@<pubkey>` form produced by repo_identity.Render) in the same
// box slot AddRepoPubKey would occupy. The value is emitted verbatim --- the
// `@` separating handle and pubkey is already part of s --- so downstream box
// parsing sees the field in the unchanged position.
func (builder *Builder) AddRepoIdentity(s string) {
	if s == "" {
		return
	}

	builder.Contents.Append(string_format_writer.FormattedField{
		Field: fields.Field{
			Value: s,
			Type:  fields.TypeHash,
		},
		NoTruncate: true,
	})
}

func (builder *Builder) AddObjectSig(
	metadata objects.MetadataMutable,
	funcAbbreviate domain_interfaces.FuncAbbreviateString,
) {
	if funcAbbreviate == nil {
		builder.addMarklId(metadata.GetObjectSig())
	} else {
		builder.addMarklIdAbbreviated(metadata.GetObjectSig(), funcAbbreviate)
	}
}

func (builder *Builder) AddMotherSigIfNecessary(
	metadata objects.MetadataMutable,
	funcAbbreviate domain_interfaces.FuncAbbreviateString,
) {
	id := metadata.GetMotherObjectSig()

	if id.IsNull() {
		return
	}

	if funcAbbreviate == nil {
		builder.addMarklId(id)
	} else {
		builder.addMarklIdAbbreviated(id, funcAbbreviate)
	}
}

func (builder *Builder) addMarklIdIfNotNull(id mad_domain_interfaces.MarklId) {
	if id.IsNull() {
		return
	}

	builder.addMarklId(id)
}

func (builder *Builder) addMarklId(id mad_domain_interfaces.MarklId) {
	builder.addMarklIdWithColorType(id, id.GetPurposeId(), fields.TypeHash)
}

func (builder *Builder) addMarklIdAbbreviated(
	id mad_domain_interfaces.MarklId,
	funcAbbreviate domain_interfaces.FuncAbbreviateString,
) {
	value := id.String()

	if funcAbbreviate != nil {
		abbreviated, err := funcAbbreviate(id)
		if err != nil {
			panic(err)
		}

		// Empty means the abbreviation index has no entries; use the
		// full id (see AddBlobDigestIfNecessary).
		if abbreviated != "" {
			value = abbreviated
		}
	}

	builder.Contents.Append(string_format_writer.FormattedField{
		Field: fields.Field{
			Key:   id.GetPurposeId(),
			Value: value,
			Type:  fields.TypeHash,
		},
		Separator:  '@',
		NoTruncate: true,
	})
}

func (builder *Builder) addMarklIdLockWithColorType(
	key string,
	value mad_domain_interfaces.MarklId,
	colorType fields.Type,
) {
	builder.addMarklIdWithColorType(value, key, colorType)
}

func (builder *Builder) addMarklIdWithColorType(
	value mad_domain_interfaces.MarklId,
	key string,
	colorType fields.Type,
) {
	if key == "" {
		panic(fmt.Sprintf("empty key for markl id: %q", value))
	}

	builder.Contents.Append(string_format_writer.FormattedField{
		Field: fields.Field{
			Key:   key,
			Value: value.String(),
			Type:  colorType,
		},
		Separator:  '@',
		NoTruncate: true,
	})
}

func (builder *Builder) AddError(err error) {
	var errorGroup errors.Group

	if errors.As(err, &errorGroup) {
		for _, err := range errorGroup {
			builder.Contents.Append(
				string_format_writer.FormattedField{
					Field: fields.Field{
						Key:   "error",
						Value: err.Error(),
						Type:  fields.TypeUserData,
					},
					NoTruncate: true,
				},
			)
		}
	} else {
		builder.Contents.Append(
			string_format_writer.FormattedField{
				Field: fields.Field{
					Key:   "error",
					Value: err.Error(),
					Type:  fields.TypeUserData,
				},
				NoTruncate: true,
			},
		)
	}
}

func (builder *Builder) AddTai(metadata objects.MetadataMutable) {
	builder.Contents.Append(string_format_writer.FormattedField{
		Field: fields.Field{
			Value: metadata.GetTai().String(),
			Type:  fields.TypeHash,
		},
	})
}

func (builder *Builder) AddType(
	metadata objects.MetadataMutable,
) {
	builder.Contents.Append(string_format_writer.FormattedField{
		Field: fields.Field{
			Value: metadata.GetType().String(),
			Type:  fields.TypeType,
		},
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
			fields.TypeType,
		)
	}
}

func (builder *Builder) AddTags(metadata objects.MetadataMutable) {
	tagCount := metadata.GetTags().Len()
	builder.Contents.Grow(tagCount)

	for tag := range metadata.AllTags() {
		builder.Contents.Append(string_format_writer.FormattedField{
			Field: fields.Field{
				Value: tag.String(),
			},
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

func (builder *Builder) AddReferencedObjectsAndLocks(metadata objects.MetadataMutable) {
	for ref := range metadata.AllReferencedObjects() {
		lockValue := metadata.GetReferencedObjectLock(ref).GetValue()
		alias := metadata.GetReferenceAlias(ref)

		var key string
		if alias != "" {
			key = alias + "<" + ref.String()
		} else {
			key = "<" + ref.String()
		}

		if lockValue.IsEmpty() {
			builder.Contents.Append(string_format_writer.FormattedField{
				Field: fields.Field{
					Value: key,
					Type:  fields.TypeId,
				},
			})
		} else {
			builder.addMarklIdLockWithColorType(
				key,
				lockValue,
				fields.TypeId,
			)
		}
	}
}

// AddBlobReferences emits each typed blob reference as TWO space-free fields
// (`alias<@digest`, then `!type@sig`), matching the doddish grammar (two seqs
// split by a space -- see 0/doddish/scanner_test.go "typed blob ref" cases)
// and the box reader (box_format/read.go TokenMatcherBlobReference* +
// scanBlobReferenceTypeLock lookahead). A single combined field would embed a
// space, which the fields writer quotes -- and a quoted string decodes as a
// DESCRIPTION, silently dropping the references and corrupting the signed
// object digest (the blob-ref round-trip bug behind repo-fsck / transfer
// signature failures).
func (builder *Builder) AddBlobReferences(metadata objects.MetadataMutable) {
	for blobId := range metadata.AllBlobReferences() {
		alias := metadata.GetBlobReferenceAlias(blobId)

		var value string
		if alias != "" {
			value = alias + "<@" + blobId.String()
		} else {
			value = "<@" + blobId.String()
		}

		builder.Contents.Append(string_format_writer.FormattedField{
			Field: fields.Field{
				Value: value,
				Type:  fields.TypeId,
			},
			NoTruncate: true,
		})

		typeLock := metadata.GetBlobReferenceTypeLock(blobId)
		typeLockStr := markl.MakeLockCoderValueNotRequired(typeLock).String()

		if typeLockStr != "" {
			builder.Contents.Append(string_format_writer.FormattedField{
				Field: fields.Field{
					Value: typeLockStr,
					Type:  fields.TypeType,
				},
				NoTruncate: true,
			})
		}
	}
}

func (builder *Builder) AddDescription(metadata objects.MetadataMutable) {
	builder.Contents.Append(string_format_writer.FormattedField{
		Field: fields.Field{
			Value: metadata.GetDescription().StringWithoutNewlines(),
			Type:  fields.TypeUserData,
		},
	})
}

// AddDescriptionPreservingNewlines writes the description's exact string
// (embedded newlines intact, %q-escaped by the field writer) instead of
// the display-oriented single-line collapse AddDescription uses. Archive/
// wire-format callers (inventory_list entries) must round-trip the exact
// bytes an object's signature was computed over; collapsing newlines
// there silently changes the signed digest on decode.
func (builder *Builder) AddDescriptionPreservingNewlines(metadata objects.MetadataMutable) {
	builder.Contents.Append(string_format_writer.FormattedField{
		Field: fields.Field{
			Value: metadata.GetDescription().String(),
			Type:  fields.TypeUserData,
		},
	})
}
