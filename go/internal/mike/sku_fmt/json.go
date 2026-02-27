package sku_fmt

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/echo/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/internal/hotel/objects"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter_set"
)

type JSONMCP struct {
	URI         string   `json:"uri,omitempty"`
	RelatedURIs []string `json:"related_uris,omitempty"`
}

type JSON struct {
	// TODO add object digest
	BlobId      string   `json:"blob-sha"`
	BlobString  string   `json:"blob-string,omitempty"`
	Date        string   `json:"date"`
	Description string   `json:"description"`
	Dormant     bool     `json:"dormant"`
	ObjectId    string   `json:"object-id"`
	RepoPubkey  markl.Id `json:"repo-pub_key"`
	RepoSig     markl.Id `json:"repo-sig"`
	Tags        []string `json:"tags"`
	Tai         string   `json:"tai"`
	Type        string   `json:"type"`

	JSONMCP
}

func (json *JSON) FromStringAndMetadata(
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
	json.Dormant = metadata.GetIndex().GetDormant().Bool()
	json.ObjectId = objectId
	json.RepoPubkey.ResetWithMarklId(metadata.GetRepoPubKey())
	json.RepoSig.ResetWithMarklId(metadata.GetObjectSig())
	json.Tags = slices.Collect(quiter_set.Strings(metadata.GetTags()))
	json.Tai = metadata.GetTai().String()
	json.Type = metadata.GetType().String()

	json.URI = fmt.Sprintf("dodder:///objects/%s", objectId)

	json.RelatedURIs = make([]string, 0, 2+len(json.Tags))

	json.RelatedURIs = append(
		json.RelatedURIs,
		fmt.Sprintf("dodder:///objects/%s", metadata.GetType()),
	)

	for _, tag := range json.Tags {
		json.RelatedURIs = append(
			json.RelatedURIs,
			fmt.Sprintf("dodder:///objects/%s", tag),
		)
	}

	// json.MCPURI = fmt.Sprintf(
	// 	"dodder://objects/%s@%s:%s",
	// 	objectId,
	// 	json.RepoPubkey,
	// 	json.RepoSig,
	// )

	// TODO add support for "preview"

	return err
}

// TODO accept blob store instead of env
func (json *JSON) FromTransacted(
	object *sku.Transacted,
	blobStore domain_interfaces.BlobStore,
) (err error) {
	return json.FromStringAndMetadata(
		object.ObjectId.String(),
		object.GetMetadataMutable(),
		blobStore,
	)
}

func (json *JSON) ToTransacted(
	object *sku.Transacted,
	blobStore domain_interfaces.BlobStore,
) (err error) {
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
		object.SetBlobDigest(writeCloser.GetMarklId())
	}

	// Set BlobId from JSON even if not writing to blob store
	if json.BlobId != "" && blobStore == nil {
		if err = object.GetMetadataMutable().GetBlobDigestMutable().Set(
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

	if err = object.GetMetadataMutable().GetTypeMutable().SetType(json.Type); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = object.GetMetadataMutable().GetDescriptionMutable().Set(json.Description); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var tagSet ids.TagSet

	if tagSet, err = ids.MakeTagSetStrings(json.Tags...); err != nil {
		err = errors.Wrap(err)
		return err
	}

	objects.SetTags(object.GetMetadataMutable(), tagSet)
	object.GetMetadataMutable().GenerateExpandedTags()

	object.GetMetadataMutable().GetRepoPubKeyMutable().ResetWithMarklId(json.RepoPubkey)
	object.GetMetadataMutable().GetObjectSigMutable().ResetWithMarklId(json.RepoSig)

	// Set Tai from either Date or Tai field
	if json.Tai != "" {
		if err = object.GetMetadataMutable().GetTaiMutable().Set(json.Tai); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else if json.Date != "" {
		if err = object.GetMetadataMutable().GetTaiMutable().SetFromRFC3339(json.Date); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	// Set Dormant state
	object.GetMetadataMutable().GetIndexMutable().GetDormantMutable().SetBool(json.Dormant)

	return err
}
