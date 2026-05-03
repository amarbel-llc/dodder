package haustoria

// Haustoria is the interface for a pluggable checkout store.
// Named after the organ through which parasitic plants (like dodder)
// tap into host organisms to exchange nutrients.
//
// A Haustoria implementation translates between dodder's internal object
// representation and an external system's format (CalDAV VTODOs, browser
// bookmarks, WebDAV files, etc.).
//
// Compile = external → dodder (like compiling source into an executable).
// Decompile = dodder → external (like decompiling back to source).
type Haustoria interface {
	// Compile reads an external resource and returns dodder-compatible fields.
	// external → dodder
	Compile(CompileRequest) (CompileResult, error)

	// Decompile writes a dodder object to the external store.
	// dodder → external
	Decompile(DecompileRequest) (DecompileResult, error)

	// Discover returns external resources in the store.
	Discover() ([]ExternalResource, error)

	// Delete removes an external resource by its external identifier.
	Delete(externalId string) error
}

// CompileRequest identifies an external resource to compile into dodder.
type CompileRequest struct {
	// ExternalId is the external system's identifier.
	ExternalId string
}

// CompileResult contains the dodder-compatible fields extracted from an
// external resource.
type CompileResult struct {
	// ExternalId is the external system's identifier.
	ExternalId string

	// Description maps to the dodder object description.
	Description string

	// Blob is the compiled blob content.
	Blob []byte

	// Tags are compiled tag identifiers.
	Tags []string

	// TypeId is the inferred dodder type (e.g. "!task" from VTODO).
	TypeId string

	// ETag is the external resource's current ETag.
	ETag string
}

// DecompileRequest contains the dodder object data to decompile to the
// external store.
type DecompileRequest struct {
	// ObjectId is the dodder object identifier.
	ObjectId string

	// Description is the object's description (maps to e.g. VTODO SUMMARY).
	Description string

	// Blob is the object's blob content (maps to e.g. VTODO DESCRIPTION).
	Blob []byte

	// Tags are the object's tag identifiers.
	Tags []string

	// TypeId is the object's type identifier (e.g. "!task").
	TypeId string

	// ExternalId is the existing external identifier, if updating.
	// Empty for new objects.
	ExternalId string

	// ETag is the last known ETag for conditional updates.
	ETag string
}

// DecompileResult contains the result of a Decompile operation.
type DecompileResult struct {
	// ExternalId is the external system's identifier for the resource.
	ExternalId string

	// ETag is the external resource's ETag after the operation.
	ETag string
}

// ExternalResource represents an external resource discovered during sync.
type ExternalResource struct {
	// ExternalId is the external system's identifier.
	ExternalId string

	// TypeId is the inferred dodder type.
	TypeId string

	// Description is a human-readable summary.
	Description string
}
