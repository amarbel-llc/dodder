package store

import (
	"slices"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/alfa/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
)

// terminalValuesByField maps a field name to the enum values its type declares
// terminal (FDR 0025).
type terminalValuesByField map[string][]string

// terminalValuesCache memoizes terminalValuesByField per type blob digest so
// the stream index's pre-write step parses each type blob at most once.
type terminalValuesCache struct {
	lock     sync.Mutex
	byDigest map[string]terminalValuesByField
}

// isInTerminalState reports whether any of the object's projected fields holds
// a value its type declares terminal. Objects without projected fields, or
// whose type can't be resolved, are never type-dormant.
func (store *Store) isInTerminalState(object *sku.Transacted) bool {
	metadata := object.GetMetadata()

	// Resolved lazily on the first field so objects without projected fields
	// never touch the type.
	var terminalValues terminalValuesByField
	resolved := false

	for field := range metadata.GetIndex().GetFields() {
		if !resolved {
			resolved = true

			typeObject := store.storeConfig.GetConfig().GetApproximatedType(
				metadata.GetType(),
			).ApproximatedOrActual()

			if typeObject == nil {
				return false
			}

			if terminalValues = store.getTerminalValues(typeObject); len(terminalValues) == 0 {
				return false
			}
		}

		if slices.Contains(terminalValues[field.Key], field.Value) {
			return true
		}
	}

	return false
}

func (store *Store) getTerminalValues(
	typeObject *sku.Transacted,
) terminalValuesByField {
	digest := typeObject.GetBlobDigest().String()

	cache := &store.terminalValuesCache
	cache.lock.Lock()
	defer cache.lock.Unlock()

	if terminalValues, ok := cache.byDigest[digest]; ok {
		return terminalValues
	}

	terminalValues, err := store.parseTerminalValues(typeObject)
	if err != nil {
		// Not cached: the failure may be transient (e.g. a type blob not yet
		// readable mid-import).
		ui.Log().Printf(
			"reading terminal values for %s: %s",
			typeObject.GetObjectId(),
			err,
		)

		return nil
	}

	if cache.byDigest == nil {
		cache.byDigest = make(map[string]terminalValuesByField)
	}

	cache.byDigest[digest] = terminalValues

	return terminalValues
}

// parseTerminalValues reads the type blob's valid terminal declarations. An
// invalid declaration is skipped here (projection rejects it on commit), so
// it never drives dormancy.
func (store *Store) parseTerminalValues(
	typeObject *sku.Transacted,
) (terminalValues terminalValuesByField, err error) {
	blob, repool, _, err := store.GetTypedBlobStore().Type.ParseTypedBlob(
		typeObject.GetType(),
		typeObject.GetBlobDigest(),
	)

	if repool != nil {
		defer repool()
	}

	if err != nil {
		return terminalValues, err
	}

	typeBlobWithFields, ok := blob.(type_blobs.WithFields)

	if !ok {
		return terminalValues, err
	}

	for _, fieldDefinition := range typeBlobWithFields.GetFieldDefinitions() {
		if len(fieldDefinition.Terminal) == 0 ||
			fieldDefinition.ValidateTerminal() != nil {
			continue
		}

		if terminalValues == nil {
			terminalValues = make(terminalValuesByField)
		}

		// Cloned: the blob returns to its pool on repool, and its slices may
		// be reused by the next decode.
		terminalValues[fieldDefinition.Name] = slices.Clone(
			fieldDefinition.Terminal,
		)
	}

	return terminalValues, err
}
