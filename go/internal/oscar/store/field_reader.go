package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/alfa/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/script_config"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func (store *Store) tryReadFields(
	daughter *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	// Required-field enforcement (including the blobless rejection below)
	// deliberately shares this gate with enum validation: both are part of
	// the script-projection machinery, so a commit with RunHooks off skips
	// them, and the caller in mutating.go swallows their errors when
	// IgnoreHookErrors is set. Field validation is a projection-level
	// constraint, not an unconditional store invariant.
	if !options.RunHooks {
		return err
	}

	if daughter.GetType().IsEmpty() {
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

	// A blobless object cannot satisfy any Required field, so reject it
	// before the fields-reader gate: a type that declares required fields
	// but ships no reader script still refuses blobless commits. Types
	// without required fields keep the old behavior (nothing to project,
	// nothing to validate).
	if daughter.GetBlobDigest().IsNull() {
		return validateBloblessAgainstRequiredFields(
			daughter.GetType(),
			fieldDefs,
		)
	}

	fieldsReader := typeBlobWithFields.GetFieldsReader()
	if fieldsReader == nil {
		return err
	}

	blobReader, err := store.GetEnvRepo().GetReadBlobStore().MakeBlobReader(
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

	return projectFields(
		daughter.GetType(),
		fieldDefs,
		scriptOutput,
		typeBlobDigest,
		func(field fields.Field) {
			fieldsMutable.Append(field)
		},
	)
}

// projectFields validates the fields-reader projection (scriptOutput)
// against the type's field definitions and appends the resulting index
// fields via appendField. A validation failure rejects the commit (subject
// to the IgnoreHookErrors handling in mutating.go).
func projectFields(
	tipe ids.Type,
	fieldDefs []type_blobs.FieldDefinition,
	scriptOutput map[string]string,
	typeBlobDigest markl.Id,
	appendField func(fields.Field),
) (err error) {
	for _, fd := range fieldDefs {
		value, ok := scriptOutput[fd.Name]
		if !ok {
			value = fd.Default
		}

		// Required rejects both absence and explicit emptiness with one
		// check: an absent key falls back to the default above (so a
		// non-empty default always satisfies Required), while a no-default
		// required field projects "" for both the missing-key and `key = ""`
		// shapes — and "" is never an acceptable required value.
		if fd.Required && value == "" {
			return errors.Wrap(fmt.Errorf(
				"type %s: field %q is required but missing or empty",
				tipe, fd.Name,
			))
		}

		// A no-default enum whose projected value is empty is optional and
		// genuinely unset: empty arises both when the blob omits the key
		// (absent) and when the actionable fields-writer materializes it as
		// `key = ""` (an unset DODDER_FIELD_* env var expands to empty), and
		// "" is never a valid enum value anyway — so skip both the enum
		// validation and the index append rather than rejecting "untriaged".
		// String fields keep empty values: an empty string is a legitimate
		// value (e.g. the no-default `due` field renders as `due=`). A field
		// that declares a default still rejects an explicit empty value.
		if fd.Kind == "enum" && value == "" && fd.Default == "" {
			continue
		}

		if fd.Kind == "enum" && len(fd.Values) > 0 {
			if !slices.Contains(fd.Values, value) {
				return errors.Wrap(fmt.Errorf(
					"field %q: value %q is not in allowed values %v",
					fd.Name, value, fd.Values,
				))
			}
		}

		appendField(fields.Field{
			Type:           fields.TypeUserData,
			Key:            fd.Name,
			Value:          value,
			TypeBlobDigest: typeBlobDigest,
		})
	}

	return err
}

// validateBloblessAgainstRequiredFields rejects a blobless commit of a type
// that declares any Required field; a type without required fields accepts
// blobless commits as before.
func validateBloblessAgainstRequiredFields(
	tipe ids.Type,
	fieldDefs []type_blobs.FieldDefinition,
) (err error) {
	if required := requiredFieldNames(fieldDefs); len(required) > 0 {
		return errors.Wrap(fmt.Errorf(
			"type %s requires fields (%s) but object has no blob",
			tipe,
			strings.Join(required, ", "),
		))
	}

	return err
}

// requiredFieldNames returns the names of field definitions marked
// Required, in definition order.
func requiredFieldNames(fieldDefs []type_blobs.FieldDefinition) []string {
	var names []string

	for _, fd := range fieldDefs {
		if fd.Required {
			names = append(names, fd.Name)
		}
	}

	return names
}
