package store

import (
	"bytes"
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/type_blobs"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/script_config"
)

type edgeExplorer struct {
	objectStore sku.RepoStore
	blobStore   domain_interfaces.BlobStore
}

func MakeEdgeExplorer(
	objectStore sku.RepoStore,
	blobStore domain_interfaces.BlobStore,
) sku.EdgeExplorer {
	return &edgeExplorer{
		objectStore: objectStore,
		blobStore:   blobStore,
	}
}

func (e *edgeExplorer) ExploreEdges(
	object *sku.Transacted,
) (edges sku.Edges, err error) {
	// 1. Type edge
	if typeId := object.GetType(); !typeId.IsEmpty() && !ids.IsBuiltin(typeId) {
		var oid ids.ObjectId

		if err = oid.SetWithId(typeId); err != nil {
			return edges, errors.Wrap(err)
		}

		edges.Objects = append(edges.Objects, oid)
	}

	// 2. Tag edges
	for tag := range object.AllTags() {
		var oid ids.ObjectId

		if err = oid.SetWithId(tag); err != nil {
			return edges, errors.Wrap(err)
		}

		edges.Objects = append(edges.Objects, oid)
	}

	// 3. Referenced object edges
	for ref := range object.GetMetadata().AllReferencedObjects() {
		refCopy := ref

		edges.Objects = append(edges.Objects, refCopy)
	}

	// 4. Blob reference edges (with transitive blob→blob traversal)
	seen := make(map[string]struct{})

	for blobDigest := range object.GetMetadata().AllBlobReferences() {
		blobCopy := blobDigest
		edges.Blobs = append(edges.Blobs, blobCopy)

		typeLock := object.GetMetadata().GetBlobReferenceTypeLock(blobDigest)

		if typeLock.GetKey().IsEmpty() {
			continue
		}

		seen[blobDigest.String()] = struct{}{}

		nestedEdges, discoverErr := e.discoverBlobEdges(blobDigest, typeLock, seen)
		if discoverErr != nil {
			edges.Skipped = append(edges.Skipped,
				errors.Wrapf(discoverErr, "blob %s", blobDigest.String()))

			continue
		}

		edges.Objects = append(edges.Objects, nestedEdges.Objects...)
		edges.Blobs = append(edges.Blobs, nestedEdges.Blobs...)
	}

	return edges, nil
}

func (e *edgeExplorer) discoverBlobEdges(
	blobDigest markl.Id,
	typeLock markl.Lock[ids.SeqId, *ids.SeqId],
	seen map[string]struct{},
) (edges sku.Edges, err error) {
	referencesConfig, err := e.getReferencesConfig(typeLock)
	if err != nil {
		return edges, errors.Wrap(err)
	}

	if referencesConfig == nil {
		return edges, nil
	}

	blobReader, err := e.blobStore.MakeBlobReader(blobDigest)
	if err != nil {
		return edges, errors.Wrap(err)
	}

	defer errors.DeferredCloser(&err, blobReader)

	var stdout io.WriterTo

	if stdout, err = script_config.MakeWriterToWithStdin(
		&referencesConfig.ScriptConfig,
		nil,
		blobReader,
	); err != nil {
		if referencesConfig.Optional {
			return edges, nil
		}

		return edges, errors.Wrap(err)
	}

	var buf bytes.Buffer

	if _, err = stdout.WriteTo(&buf); err != nil {
		if referencesConfig.Optional {
			return edges, nil
		}

		return edges, errors.Wrap(err)
	}

	var refs []discoveredReference

	if refs, err = parseReferenceOutput(buf.String()); err != nil {
		return edges, errors.Wrap(err)
	}

	for _, ref := range refs {
		if ref.BlobId != "" {
			var id markl.Id

			if err = id.Set(ref.BlobId); err != nil {
				return edges, errors.Wrapf(err, "invalid blob ref: %q", ref.BlobId)
			}

			edges.Blobs = append(edges.Blobs, id)

			blobKey := id.String()

			if _, alreadySeen := seen[blobKey]; alreadySeen {
				continue
			}

			seen[blobKey] = struct{}{}

			if ref.TypeId == "" {
				continue
			}

			var nestedTypeLock markl.Lock[ids.SeqId, *ids.SeqId]

			marshaler := markl.MakeMutableLockCoderValueNotRequired(
				&nestedTypeLock,
			)

			if err = marshaler.Set(ids.MakeTypeString(ref.TypeId)); err != nil {
				edges.Skipped = append(edges.Skipped,
					errors.Wrapf(err, "blob %s type lock %q", blobKey, ref.TypeId))

				continue
			}

			nestedEdges, discoverErr := e.discoverBlobEdges(
				id,
				nestedTypeLock,
				seen,
			)

			if discoverErr != nil {
				edges.Skipped = append(edges.Skipped,
					errors.Wrapf(discoverErr, "blob %s", blobKey))

				continue
			}

			edges.Objects = append(edges.Objects, nestedEdges.Objects...)
			edges.Blobs = append(edges.Blobs, nestedEdges.Blobs...)
			edges.Skipped = append(edges.Skipped, nestedEdges.Skipped...)
		} else if ref.ObjectId != "" {
			var oid ids.ObjectId

			if err = oid.Set(ref.ObjectId); err != nil {
				return edges, errors.Wrapf(err, "invalid object ref: %q", ref.ObjectId)
			}

			edges.Objects = append(edges.Objects, oid)
		}
	}

	return edges, nil
}

func (e *edgeExplorer) getReferencesConfig(
	typeLock markl.Lock[ids.SeqId, *ids.SeqId],
) (config *type_blobs.ReferencesConfig, err error) {
	typeId := typeLock.GetKey()

	var typeOid ids.ObjectId

	if err = typeOid.SetWithId(&typeId); err != nil {
		return nil, errors.Wrap(err)
	}

	fetched, repool := sku.GetTransactedPool().GetWithRepool()
	defer repool()

	if err = e.objectStore.ReadOneInto(&typeOid, fetched); err != nil {
		return nil, errors.Wrap(err)
	}

	if fetched.GetBlobDigest().IsNull() {
		return nil, nil
	}

	blobReader, err := e.blobStore.MakeBlobReader(fetched.GetBlobDigest())
	if err != nil {
		return nil, errors.Wrap(err)
	}

	defer errors.DeferredCloser(&err, blobReader)

	tipeString := fetched.GetType().String()
	if tipeString == "" {
		tipeString = ids.TypeTomlTypeV0
	}

	typedBlob := type_blobs.TypedBlob{
		Type: ids.MustTypeStruct(tipeString),
	}

	bufferedReader, repoolBufferedReader := pool.GetBufferedReader(blobReader)
	defer repoolBufferedReader()

	if _, err = type_blobs.CoderToTypedBlob.Blob.DecodeFrom(
		&typedBlob,
		bufferedReader,
	); err != nil {
		return nil, errors.Wrap(err)
	}

	return typedBlob.Blob.GetReferences(), nil
}
