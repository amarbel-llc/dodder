package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/alfa/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/script_config"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

func (store *Store) tryReadFields(
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

	typeBlobWithFields, ok := blob.(type_blobs.WithFields)
	if !ok {
		return err
	}

	fieldDefs := typeBlobWithFields.GetFieldDefinitions()
	if len(fieldDefs) == 0 {
		return err
	}

	fieldsReader := typeBlobWithFields.GetFieldsReader()
	if fieldsReader == nil {
		return err
	}

	blobReader, err := store.GetEnvRepo().GetDefaultBlobStore().MakeBlobReader(
		daughter.GetBlobDigest(),
	)
	if err != nil {
		return errors.Wrap(err)
	}

	defer errors.DeferredCloser(&err, blobReader)

	env := make(interfaces.EnvVars)
	store.GetEnvRepo().AddToEnvVars(env)

	var stdout io.WriterTo

	if stdout, err = script_config.MakeWriterToWithStdin(
		fieldsReader,
		env,
		blobReader,
	); err != nil {
		return errors.Wrap(err)
	}

	var buf bytes.Buffer

	if _, err = stdout.WriteTo(&buf); err != nil {
		return errors.Wrap(err)
	}

	var scriptOutput map[string]string

	if err = json.Unmarshal(buf.Bytes(), &scriptOutput); err != nil {
		return errors.Wrapf(err, "fields-reader script output is not valid JSON")
	}

	var typeBlobDigest markl.Id
	typeBlobDigest.ResetWithMarklId(typeObject.GetBlobDigest())

	fieldsMutable := daughter.GetMetadataMutable().GetIndexMutable().GetFieldsMutable()

	for _, fd := range fieldDefs {
		value, ok := scriptOutput[fd.Name]
		if !ok {
			value = fd.Default
		}

		if fd.Kind == "enum" && len(fd.Values) > 0 {
			if !slices.Contains(fd.Values, value) {
				return errors.Wrap(fmt.Errorf(
					"field %q: value %q is not in allowed values %v",
					fd.Name, value, fd.Values,
				))
			}
		}

		fieldsMutable.Append(fields.Field{
			Type:           fields.TypeUserData,
			Key:            fd.Name,
			Value:          value,
			TypeBlobDigest: typeBlobDigest,
		})
	}

	return err
}
