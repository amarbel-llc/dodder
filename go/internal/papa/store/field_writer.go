package store

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/type_blobs"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func (store *Store) tryWriteFields(
	daughter *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	if !options.RunHooks {
		return err
	}

	if daughter.GetBlobDigest().IsNull() {
		return err
	}

	// Check if the object has any type-defined fields (non-empty TypeBlobDigest)
	hasTypeDefinedFields := false

	for field := range daughter.GetMetadata().GetIndex().GetFields() {
		if !field.TypeBlobDigest.IsNull() {
			hasTypeDefinedFields = true
			break
		}
	}

	if !hasTypeDefinedFields {
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

	typeBlobWithFields, ok := blob.(type_blobs.WithFields)
	if !ok {
		return err
	}

	fieldsWriter := typeBlobWithFields.GetFieldsWriter()
	if fieldsWriter == nil {
		return err
	}

	// Read the current blob content
	blobReader, err := store.GetEnvRepo().GetDefaultBlobStore().MakeBlobReader(
		daughter.GetBlobDigest(),
	)
	if err != nil {
		return errors.Wrap(err)
	}

	var blobContent bytes.Buffer

	if _, err = io.Copy(&blobContent, blobReader); err != nil {
		errors.DeferredCloser(&err, blobReader)
		return errors.Wrap(err)
	}

	if err = blobReader.Close(); err != nil {
		return errors.Wrap(err)
	}

	// Write blob content to a temp file
	tmpFile, err := os.CreateTemp("", "dodder-field-writer-*")
	if err != nil {
		return errors.Wrap(err)
	}

	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err = tmpFile.Write(blobContent.Bytes()); err != nil {
		tmpFile.Close()
		return errors.Wrap(err)
	}

	if err = tmpFile.Close(); err != nil {
		return errors.Wrap(err)
	}

	// Build environment variables: DODDER_BLOB_PATH + DODDER_FIELD_<name>=<value>
	env := make(map[string]string)
	env["DODDER_BLOB_PATH"] = tmpPath

	for field := range daughter.GetMetadata().GetIndex().GetFields() {
		if !field.TypeBlobDigest.IsNull() {
			env[fmt.Sprintf("DODDER_FIELD_%s", field.Key)] = field.Value
		}
	}

	// Execute the fields-writer script
	cmd, err := fieldsWriter.Cmd()
	if err != nil {
		return errors.Wrap(err)
	}

	envCollapsed := os.Environ()

	for k, v := range env {
		envCollapsed = append(envCollapsed, fmt.Sprintf("%s=%s", k, v))
	}

	cmd.Env = envCollapsed

	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "fields-writer script failed: %s", string(output))
	}

	// Read the modified temp file back
	modifiedContent, err := os.ReadFile(tmpPath)
	if err != nil {
		return errors.Wrap(err)
	}

	// Write the new blob content to the blob store
	blobWriter, err := store.GetEnvRepo().GetDefaultBlobStore().MakeBlobWriter(nil)
	if err != nil {
		return errors.Wrap(err)
	}

	defer errors.DeferredCloser(&err, blobWriter)

	if _, err = bytes.NewReader(modifiedContent).WriteTo(blobWriter); err != nil {
		return errors.Wrap(err)
	}

	// Update the object's blob digest with the new digest
	if err = daughter.SetBlobDigest(blobWriter.GetMarklId()); err != nil {
		return errors.Wrap(err)
	}

	// Clear existing fields so tryReadFields will re-project them
	daughter.GetMetadataMutable().GetIndexMutable().GetFieldsMutable().Reset()

	return err
}
