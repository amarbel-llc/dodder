package sku_json_fmt

import (
	"io"
	"slices"
	"strings"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/0/quiter_seq"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
)

type Transacted struct {
	AbbreviatedObjectId string                   `json:"abbreviated-object-id,omitempty"`
	BlobId              string                   `json:"blob-id"`
	BlobReferences      map[string]BlobReference `json:"blob-references,omitempty"`
	BlobString          string                   `json:"blob-string,omitempty"`
	Date                string                   `json:"date"`
	Description         string                   `json:"description"`
	Lock                Lock                     `json:"lock"`
	MotherObjectSig     markl.Id                 `json:"mother-object-sig"`
	ObjectDigest        markl.Id                 `json:"object-digest"`
	ObjectId            string                   `json:"object-id"`
	RepoPubkey          markl.Id                 `json:"repo-pub_key"`
	RepoSig             markl.Id                 `json:"repo-sig"`
	Sha                 string                   `json:"sha"`
	Tags                []string                 `json:"tags"`
	Tai                 string                   `json:"tai"`
	Type                string                   `json:"type"`
}

// TODO make a json factory

func (json *Transacted) FromObjectIdStringAndMetadata(
	objectId string,
	metadata objects.MetadataMutable,
	blobStore mad_domain_interfaces.BlobStore,
) (err error) {
	if blobStore != nil {
		var readCloser mad_domain_interfaces.BlobReader

		if readCloser, err = blobStore.MakeBlobReader(
			metadata.GetBlobDigest(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloser(&err, readCloser)

		var blobStringBuilder strings.Builder

		if _, err = io.Copy(&blobStringBuilder, readCloser); err != nil {
			err = errors.Wrap(err)
			return err
		}

		json.BlobString = blobStringBuilder.String()
	}

	json.BlobId = metadata.GetBlobDigest().String()
	json.Date = metadata.GetTai().Format(string_format_writer.StringFormatDateTime)
	json.Description = metadata.GetDescription().String()
	json.MotherObjectSig.ResetWithMarklId(metadata.GetMotherObjectSig())
	json.ObjectDigest.ResetWithMarklId(metadata.GetObjectDigest())
	json.ObjectId = objectId
	json.RepoPubkey.ResetWithMarklId(metadata.GetRepoPubKey())
	json.RepoSig.ResetWithMarklId(metadata.GetObjectSig())
	json.Tags = slices.Collect(quiter.Strings(quiter_seq.Seq[interfaces.Collection[ids.TagStruct]](metadata.GetTags())))
	json.Tai = metadata.GetTai().String()
	json.Type = metadata.GetType().String()

	json.Lock = Lock{
		Type: metadata.GetTypeLock().GetValue().String(),
	}

	for tag := range metadata.AllTags() {
		tagLock := metadata.GetTagLock(tag)
		if tagLock != nil && !tagLock.GetValue().IsEmpty() {
			if json.Lock.Tags == nil {
				json.Lock.Tags = make(map[string]string)
			}
			json.Lock.Tags[tag.String()] = tagLock.GetValue().String()
		}
	}

	for ref := range metadata.AllReferencedObjects() {
		lock := metadata.GetReferencedObjectLock(ref)
		if lock != nil && !lock.GetValue().IsEmpty() {
			if json.Lock.References == nil {
				json.Lock.References = make(map[string]string)
			}
			json.Lock.References[ref.String()] = lock.GetValue().String()
		}

		alias := metadata.GetReferenceAlias(ref)
		if alias != "" {
			if json.Lock.ReferenceAliases == nil {
				json.Lock.ReferenceAliases = make(map[string]string)
			}
			json.Lock.ReferenceAliases[ref.String()] = alias
		}
	}

	for blobId := range metadata.AllBlobReferences() {
		blobRef := BlobReference{
			Alias: metadata.GetBlobReferenceAlias(blobId),
		}

		typeLock := metadata.GetBlobReferenceTypeLock(blobId)
		if !typeLock.IsEmpty() {
			blobRef.TypeLockKey = typeLock.GetKey().String()
			blobRef.TypeLockValue = typeLock.GetValue().String()
		}

		if blobRef != (BlobReference{}) {
			if json.BlobReferences == nil {
				json.BlobReferences = make(map[string]BlobReference)
			}
			json.BlobReferences[blobId.String()] = blobRef
		}
	}

	// TODO add support for "preview"

	return err
}

func (json *Transacted) FromTransacted(
	object *sku.Transacted,
	blobStore mad_domain_interfaces.BlobStore,
) (err error) {
	return json.FromObjectIdStringAndMetadata(
		object.GetObjectId().String(),
		object.GetMetadataMutable(),
		blobStore,
	)
}

// SetAbbreviatedObjectId populates AbbreviatedObjectId with the short form
// of the object's zettel id, using abbr's zettel abbreviator. It is a
// separate step from FromTransacted (rather than a parameter on it) so the
// many existing FromTransacted call sites that don't have an ids.Abbr handy
// are unaffected. A no-op for non-zettel genres or when abbreviation does
// not shorten the id (e.g. the abbreviation index has no entries yet), so
// AbbreviatedObjectId stays its zero value and is omitted from the JSON
// (`omitempty`) rather than duplicating ObjectId.
func (json *Transacted) SetAbbreviatedObjectId(
	abbr ids.Abbr,
	object *sku.Transacted,
) (err error) {
	if abbr.ZettelId.Abbreviate == nil {
		return err
	}

	objectId := ids.MustObjectId(object.GetObjectId())

	if err = abbr.AbbreviateZettelIdOnly(objectId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if abbreviated := objectId.String(); abbreviated != json.ObjectId {
		json.AbbreviatedObjectId = abbreviated
	}

	return err
}

func (json *Transacted) ToTransacted(
	object *sku.Transacted,
	blobStore mad_domain_interfaces.BlobStore,
) (err error) {
	metadata := object.GetMetadataMutable()

	if blobStore != nil {
		var writeCloser mad_domain_interfaces.BlobWriter

		if writeCloser, err = blobStore.MakeBlobWriter(nil); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloser(&err, writeCloser)

		reader, repool := pool.GetStringReader(json.BlobString)
		defer repool()

		if _, err = io.Copy(writeCloser, reader); err != nil {
			err = errors.Wrap(err)
			return err
		}

		// TODO just compare blob digests
		// TODO-P1 support states of blob vs blob sha
		markl.SetDigester(
			metadata.GetBlobDigestMutable(),
			writeCloser,
		)
	}

	// Set BlobId from JSON even if not writing to blob store
	if json.BlobId != "" && blobStore == nil {
		if err = metadata.GetBlobDigestMutable().Set(
			json.BlobId,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if err = object.GetObjectIdMutable().Set(json.ObjectId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// TODO enforce non-empty types
	if json.Type != "" {
		if err = metadata.GetTypeMutable().SetType(json.Type); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if err = metadata.GetDescriptionMutable().Set(json.Description); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var tagSet ids.TagSet

	if tagSet, err = ids.MakeTagSetStrings(json.Tags...); err != nil {
		err = errors.Wrap(err)
		return err
	}

	objects.SetTags(metadata, tagSet)
	metadata.GenerateExpandedTags()

	if !json.MotherObjectSig.IsNull() {
		metadata.GetMotherObjectSigMutable().ResetWithMarklId(json.MotherObjectSig)
	}

	if !json.ObjectDigest.IsNull() {
		metadata.GetObjectDigestMutable().ResetWithMarklId(json.ObjectDigest)
	}

	if !json.RepoPubkey.IsNull() {
		metadata.GetRepoPubKeyMutable().ResetWithMarklId(json.RepoPubkey)
	}

	if !json.RepoSig.IsNull() {
		metadata.GetObjectSigMutable().ResetWithMarklId(json.RepoSig)
	}

	if json.Lock.Type != "" {
		if err = metadata.GetTypeLockMutable().GetValueMutable().Set(
			json.Lock.Type,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	for tagStr, sigStr := range json.Lock.Tags {
		var tag ids.TagStruct
		if err = tag.Set(tagStr); err != nil {
			err = errors.Wrap(err)
			return err
		}

		tagLock := metadata.GetTagLockMutable(&tag)
		if err = tagLock.GetValueMutable().Set(sigStr); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	for refIdStr, sigStr := range json.Lock.References {
		var refId ids.SeqId
		if err = refId.Set(refIdStr); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = metadata.AddReference(refId); err != nil {
			err = errors.Wrap(err)
			return err
		}

		refLock := metadata.GetReferencedObjectLockMutable(refId)
		if err = refLock.GetValueMutable().Set(sigStr); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	for refIdStr, alias := range json.Lock.ReferenceAliases {
		var refId ids.SeqId
		if err = refId.Set(refIdStr); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = metadata.SetReferenceAlias(refId, alias); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	for blobIdStr, blobRef := range json.BlobReferences {
		var blobId markl.Id
		if err = blobId.Set(blobIdStr); err != nil {
			err = errors.Wrap(err)
			return err
		}

		var typeLock markl.Lock[ids.SeqId, *ids.SeqId]

		if blobRef.TypeLockKey != "" {
			if err = typeLock.GetKeyMutable().SetType(blobRef.TypeLockKey); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		if blobRef.TypeLockValue != "" {
			if err = typeLock.GetValueMutable().Set(blobRef.TypeLockValue); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		metadata.AddBlobReference(blobId, typeLock)

		if blobRef.Alias != "" {
			if err = metadata.SetBlobReferenceAlias(blobId, blobRef.Alias); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}
	}

	// Set Tai from either Date or Tai field
	if json.Tai != "" {
		if err = metadata.GetTaiMutable().Set(json.Tai); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else if json.Date != "" {
		if err = metadata.GetTaiMutable().SetFromRFC3339(json.Date); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}
