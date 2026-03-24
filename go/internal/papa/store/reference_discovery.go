package store

import (
	"bytes"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/type_blobs"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/script_config"
)

type discoveredReference struct {
	ObjectId string
	BlobId   string
	TypeId   string
	Alias    string
}

func parseReferenceOutput(output string) ([]discoveredReference, error) {
	var refs []discoveredReference

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var ref discoveredReference
		var value string

		if before, after, ok := strings.Cut(line, " = "); ok {
			ref.Alias = strings.TrimSpace(before)
			value = strings.TrimSpace(after)
		} else {
			value = line
		}

		if after, ok := strings.CutPrefix(value, "@"); ok {
			blobValue := after

			// Split off optional !type suffix
			if spaceIdx := strings.Index(blobValue, " "); spaceIdx != -1 {
				typeStr := strings.TrimSpace(blobValue[spaceIdx+1:])

				if strings.HasPrefix(typeStr, "!") {
					ref.TypeId = typeStr
				}

				blobValue = blobValue[:spaceIdx]
			}

			ref.BlobId = blobValue
		} else {
			ref.ObjectId = value
		}

		refs = append(refs, ref)
	}

	return refs, nil
}

func (store *Store) discoverReferences(
	daughter *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	if !options.RunHooks {
		return err
	}

	if daughter.GetBlobDigest().IsNull() {
		return err
	}

	var typeObject *sku.Transacted

	if typeObject, err = store.ReadObjectTypeAndLockIfNecessary(
		daughter,
	); err != nil {
		if errors.IsErrNotFound(err) {
			err = nil
		}

		return err
	} else if typeObject == nil {
		return err
	}

	var blob type_blobs.Blob

	{
		var repool interfaces.FuncRepool

		if blob, repool, _, err = store.GetTypedBlobStore().Type.ParseTypedBlob(
			typeObject.GetType(),
			typeObject.GetBlobDigest(),
		); err != nil {
			if repool != nil {
				repool()
			}

			return errors.Wrap(err)
		}

		defer repool()
	}

	objectReferences := blob.GetReferences()
	if objectReferences == nil {
		return err
	}

	blobReader, err := store.GetEnvRepo().GetDefaultBlobStore().MakeBlobReader(
		daughter.GetBlobDigest(),
	)
	if err != nil {
		return errors.Wrap(err)
	}

	defer errors.DeferredCloser(&err, blobReader)

	var stdout io.WriterTo

	if stdout, err = script_config.MakeWriterToWithStdin(
		&objectReferences.ScriptConfig,
		nil,
		blobReader,
	); err != nil {
		if objectReferences.Optional {
			return nil
		}

		return errors.Wrap(err)
	}

	var buf bytes.Buffer

	if _, err = stdout.WriteTo(&buf); err != nil {
		if objectReferences.Optional {
			return nil
		}

		return errors.Wrap(err)
	}

	var refs []discoveredReference

	if refs, err = parseReferenceOutput(buf.String()); err != nil {
		return errors.Wrap(err)
	}

	metadata := daughter.GetMetadataMutable()

	for _, ref := range refs {
		if ref.BlobId != "" {
			var blobId markl.Id

			if err = blobId.Set(ref.BlobId); err != nil {
				if objectReferences.Optional {
					continue
				}

				return errors.Wrapf(err, "invalid blob reference: %q", ref.BlobId)
			}

			var typeLock markl.Lock[ids.SeqId, *ids.SeqId]

			if ref.TypeId != "" {
				marshaler := markl.MakeMutableLockCoderValueNotRequired(&typeLock)

				if err = marshaler.Set(ids.MakeTypeString(ref.TypeId)); err != nil {
					if objectReferences.Optional {
						continue
					}

					return errors.Wrapf(err, "invalid blob reference type: %q", ref.TypeId)
				}
			}

			metadata.AddBlobReference(blobId, typeLock)

			if ref.Alias != "" {
				if err = metadata.SetBlobReferenceAlias(blobId, ref.Alias); err != nil {
					return errors.Wrap(err)
				}
			}
		} else {
			var refId ids.SeqId

			if err = refId.Set(ref.ObjectId); err != nil {
				if objectReferences.Optional {
					continue
				}

				return errors.Wrapf(err, "invalid reference: %q", ref.ObjectId)
			}

			if err = metadata.AddReference(refId); err != nil {
				return errors.Wrap(err)
			}

			if ref.Alias != "" {
				if err = metadata.SetReferenceAlias(refId, ref.Alias); err != nil {
					return errors.Wrap(err)
				}
			}
		}
	}

	return err
}
