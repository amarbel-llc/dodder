package haustoria

// Haustoria is the interface for a pluggable checkout store.
// Named after the organ through which parasitic plants (like dodder)
// tap into host organisms to exchange nutrients.
//
// A Haustoria implementation translates between dodder's internal object
// representation and an external system's format (CalDAV VTODOs, browser
// bookmarks, WebDAV files, etc.). Checkout is compilation (dodder → external);
// checkin is decompilation (external → dodder).
type Haustoria interface {
	// Compile writes a dodder object to the external store.
	// Returns the external identifier (e.g. CalDAV UID).
	Compile(CompileRequest) (CompileResult, error)

	// Decompile reads an external resource and returns dodder-compatible fields.
	Decompile(DecompileRequest) (DecompileResult, error)

	// Discover returns external resources that have no dodder binding
	// (created externally since last sync).
	Discover() ([]ExternalResource, error)

	// Delete removes an external resource by its external identifier.
	Delete(externalId string) error

	// Status returns a read-only summary of the external store's state.
	Status() (StatusResult, error)
}

// StatusResult summarizes the external store for display in `dodder status`.
type StatusResult struct {
	// StoreType identifies the haustoria implementation (e.g. "caldav").
	StoreType string

	// ExternalResources lists all resources in the external store.
	ExternalResources []ExternalResource
}

// CompileRequest contains the dodder object data to compile to the external
// store.
type CompileRequest struct {
	// ObjectId is the dodder object identifier.
	ObjectId string

	// Description is the object's description (maps to e.g. VTODO SUMMARY).
	Description string

	// Blob is the object's blob content (maps to e.g. VTODO DESCRIPTION).
	Blob []byte

	// Tags are the object's tag identifiers.
	Tags []string

	// TypeId is the object's type identifier (e.g. "task", "event").
	TypeId string

	// ExternalId is the existing external identifier, if updating.
	// Empty for new objects.
	ExternalId string

	// ETag is the last known ETag for conditional updates.
	ETag string
}

// CompileResult contains the result of a Compile operation.
type CompileResult struct {
	// ExternalId is the external system's identifier for the resource.
	ExternalId string

	// ETag is the external resource's ETag after the operation.
	ETag string
}

// DecompileRequest identifies an external resource to decompile.
type DecompileRequest struct {
	// ExternalId is the external system's identifier.
	ExternalId string
}

// DecompileResult contains the dodder-compatible fields extracted from an
// external resource.
type DecompileResult struct {
	// ExternalId is the external system's identifier.
	ExternalId string

	// Description maps to the dodder object description.
	Description string

	// Blob is the decompiled blob content.
	Blob []byte

	// Tags are decompiled tag identifiers.
	Tags []string

	// TypeId is the inferred dodder type (e.g. "task" from VTODO).
	TypeId string

	// ETag is the external resource's current ETag.
	ETag string
}

// ExternalResource represents an external resource discovered during sync
// that has no corresponding dodder object.
type ExternalResource struct {
	// ExternalId is the external system's identifier.
	ExternalId string

	// TypeId is the inferred dodder type.
	TypeId string

	// Description is a human-readable summary.
	Description string
}
