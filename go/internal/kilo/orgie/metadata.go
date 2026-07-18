package orgie

import (
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/0/hyphence"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/ohio"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter_set"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/format"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
)

type TagSetGetter interface {
	GetTags() ids.TagSet
}

func NewMetadata(repoId ids.RepoId) Metadata {
	return Metadata{
		RepoId:           repoId,
		TagSet:           ids.MakeTagSetFromSlice(),
		OptionCommentSet: MakeOptionCommentSet(nil),
	}
}

func NewMetadataWithOptionCommentLookup(
	repoId ids.RepoId,
	elements map[string]OptionComment,
) Metadata {
	return Metadata{
		RepoId:           repoId,
		TagSet:           ids.MakeTagSetFromSlice(),
		OptionCommentSet: MakeOptionCommentSet(elements),
	}
}

// TODO replace with embedded *sku.Transacted
type Metadata struct {
	ids.TagSet
	Matchers interfaces.Set[sku.Query] // TODO remove
	OptionCommentSet
	Type   ids.TypeStruct
	RepoId ids.RepoId
}

func (metadata *Metadata) GetTags() ids.TagSet {
	return metadata.TagSet
}

func (metadata *Metadata) SetFromObjectMetadata(
	otherMetadata objects.MetadataMutable,
	repoId ids.RepoId,
) (err error) {
	metadata.TagSet = ids.CloneTagSet(otherMetadata.GetTags())

	for comment := range otherMetadata.GetIndex().GetComments() {
		if err = metadata.OptionCommentSet.Set(comment); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	metadata.Type = otherMetadata.GetType().ToType()

	return err
}

func (metadata Metadata) RemoveFromTransacted(object sku.SkuType) (err error) {
	tags := ids.CloneTagSetMutable(object.GetSkuExternal().GetMetadata().GetTags())

	for element := range metadata.All() {
		quiter_set.Del(tags, element)
	}

	objects.SetTags(object.GetSkuExternal().GetMetadataMutable(), tags)

	return err
}

func (metadata Metadata) AsMetadata() (m1 objects.MetadataMutable) {
	m1 = objects.Make()
	m1.GetTypeMutable().ResetWithObjectId(metadata.Type)
	objects.SetTags(m1, metadata.TagSet)
	return m1
}

func (metadata Metadata) GetMetadataWriterTo() hyphence.MetadataWriterTo {
	return metadata
}

func (metadata Metadata) HasMetadataContent() bool {
	if metadata.Len() > 0 {
		return true
	}

	if !metadata.Type.IsEmpty() {
		return true
	}

	if len(metadata.OptionCommentSet.OptionComments) > 0 {
		return true
	}

	return false
}

func (metadata *Metadata) ReadFrom(reader io.Reader) (n int64, err error) {
	bufferedReader, repool := pool.GetBufferedReader(reader)
	defer repool()

	tagSet := ids.MakeTagSetMutable()
	addTag := quiter.MakeFuncAddString(tagSet)

	// `_`-reserved settings fields (dodder#374, cutting-garden RFC 0015):
	// `- _key=value` is the settings-field spelling. `% key:value` (the
	// OptionCommentSet path above) remains accepted as a deprecated
	// alias during migration. Any other `-` line (no `=`, or an
	// unreserved key) is unaffected and still parses as a tag.
	addTagOrSettingsField := func(v string) (err error) {
		if key, value, ok := strings.Cut(v, "="); ok && strings.HasPrefix(key, "_") {
			if err = metadata.OptionCommentSet.Set(
				strings.TrimPrefix(key, "_") + ":" + value,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		}

		return addTag(v)
	}

	if n, err = format.ReadLines(
		bufferedReader,
		ohio.MakeLineReaderRepeat(
			ohio.MakeLineReaderKeyValues(
				map[string]interfaces.FuncSetString{
					"%": metadata.OptionCommentSet.Set,
					"-": addTagOrSettingsField,
					"!": metadata.Type.Set,
				},
			),
		),
	); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	metadata.TagSet = ids.CloneTagSet(tagSet)

	return n, err
}

func (metadata Metadata) WriteTo(w1 io.Writer) (n int64, err error) {
	w := format.NewLineWriter()

	for _, o := range metadata.OptionCommentSet.OptionComments {
		// `_`-reserved settings fields (dodder#374, cutting-garden RFC
		// 0015): behavior-affecting settings are written as `- _key=value`
		// document fields, not `%` comments, so they aren't opaque per
		// hyphence RFC 0001. Only dry-run is migrated so far; other
		// OptionComments still use the comment spelling.
		if ocwk, ok := o.(OptionCommentWithKey); ok {
			if _, isDryRun := ocwk.OptionComment.(*OptionCommentDryRun); isDryRun {
				w.WriteFormat("- _%s=%s", ocwk.Key, ocwk.OptionComment)
				continue
			}
		}

		w.WriteFormat("%% %s", o)
	}

	for _, e := range quiter.SortedStrings(metadata.TagSet) {
		w.WriteFormat("- %s", e)
	}

	tString := metadata.Type.StringSansOp()

	if tString != "" {
		w.WriteFormat("! %s", tString)
	}

	if metadata.Matchers != nil {
		for _, c := range quiter.SortedStrings(metadata.Matchers) {
			w.WriteFormat("%% Matcher:%s", c)
		}
	}

	return w.WriteTo(w1)
}
