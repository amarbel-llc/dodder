package sku_json_fmt

import (
	"io"
	"slices"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter_seq"
	"code.linenisgreat.com/dodder/go/lib/bravo/quiter"
	"code.linenisgreat.com/dodder/go/internal/delta/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/objects"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

type Transacted struct {
	BlobId          string   `json:"blob-id"`
	BlobString      string   `json:"blob-string,omitempty"`
	Date            string   `json:"date"`
	Description     string   `json:"description"`
	Lock            Lock     `json:"lock"`
	MotherObjectSig markl.Id `json:"mother-object-sig"`
	ObjectDigest    markl.Id `json:"object-digest"`
	ObjectId        string   `json:"object-id"`
	RepoPubkey      markl.Id `json:"repo-pub_key"`
	RepoSig         markl.Id `json:"repo-sig"`
	Sha             string   `json:"sha"`
	Tags            []string `json:"tags"`
	Tai             string   `json:"tai"`
	Type            string   `json:"type"`
}

// TODO make a json factory

func (json *Transacted) FromObjectIdStringAndMetadata(
	objectId string,
	metadata objects.MetadataMutable,
	blobStore domain_interfaces.BlobStore,
) (err error) {
	if blobStore != nil {
		var readCloser domain_interfaces.BlobReader

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

	// TODO add support for "preview"

	return err
}

func (json *Transacted) FromTransacted(
	object *sku.Transacted,
	blobStore domain_interfaces.BlobStore,
) (err error) {
	return json.FromObjectIdStringAndMetadata(
		object.ObjectId.String(),
		object.GetMetadataMutable(),
		blobStore,
	)
}

func (json *Transacted) ToTransacted(
	object *sku.Transacted,
	blobStore domain_interfaces.BlobStore,
) (err error) {
	metadata := object.GetMetadataMutable()

	if blobStore != nil {
		var writeCloser domain_interfaces.BlobWriter

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

	if err = object.ObjectId.Set(json.ObjectId); err != nil {
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
