package store

import (
	"bytes"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/type_blobs"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/script_config"
)

type discoveredReference struct {
	ObjectId string
	Alias    string
}

func parseReferenceOutput(output string) ([]discoveredReference, error) {
	var refs []discoveredReference

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var ref discoveredReference

		if idx := strings.Index(line, " = "); idx != -1 {
			ref.Alias = strings.TrimSpace(line[:idx])
			ref.ObjectId = strings.TrimSpace(line[idx+3:])
		} else {
			ref.ObjectId = line
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
			return errors.Wrap(err)
		}

		defer repool()
	}

	objectReferences := blob.GetObjectReferences()
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

	metadataStruct := daughter.GetMetadataMutable().(*objects.MetadataStruct)

	for _, ref := range refs {
		var refId ids.SeqId

		if err = refId.Set(ref.ObjectId); err != nil {
			if objectReferences.Optional {
				continue
			}

			return errors.Wrapf(err, "invalid reference: %q", ref.ObjectId)
		}

		if err = metadataStruct.References.Add(refId); err != nil {
			return errors.Wrap(err)
		}

		if ref.Alias != "" {
			for index := range metadataStruct.References {
				entry := &metadataStruct.References[index]

				if entry.GetKey().String() == refId.String() {
					if err = entry.Alias.Set(ref.Alias); err != nil {
						return errors.Wrap(err)
					}

					break
				}
			}
		}
	}

	return err
}
