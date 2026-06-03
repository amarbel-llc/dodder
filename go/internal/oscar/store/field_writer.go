package store

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/alfa/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func (store *Store) tryWriteFields(
	daughter *sku.Transacted,
	mother *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	if !options.RunHooks {
		return err
	}

	if daughter.GetBlobDigest().IsNull() {
		return err
	}

	// Collect field values from daughter (the external/edited fork).
	// If fields lack TypeBlobDigest (e.g., parsed from organize text),
	// inherit it from the mother (the internal fork).
	daughterFields := make(map[string]fields.Field)

	for field := range daughter.GetMetadata().GetIndex().GetFields() {
		daughterFields[field.Key] = field
	}

	if len(daughterFields) == 0 {
		return err
	}

	// Check if any daughter field already has TypeBlobDigest.
	// If not, check mother for type-defined fields to inherit from.
	hasTypeInfo := false

	for _, field := range daughterFields {
		if !field.TypeBlobDigest.IsEmpty() {
			hasTypeInfo = true
			break
		}
	}

	if !hasTypeInfo && mother != nil {
		for field := range mother.GetMetadata().GetIndex().GetFields() {
			if field.TypeBlobDigest.IsEmpty() {
				continue
			}

			if df, ok := daughterFields[field.Key]; ok {
				df.TypeBlobDigest = field.TypeBlobDigest
				daughterFields[field.Key] = df
				hasTypeInfo = true
			}
		}
	}

	if !hasTypeInfo {
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
	blobReader, err := store.GetEnvRepo().GetReadBlobStore().MakeBlobReader(
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
	defer os.Remove(tmpPath) //defer:err-checked

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

	for _, field := range daughterFields {
		if !field.TypeBlobDigest.IsEmpty() {
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
