package blob_stores

import (
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/hotel/tap_diagnostics"
	tap "github.com/amarbel-llc/tap-dancer/go"
)

// PackOptions controls the behavior of the Pack operation.
type PackOptions struct {
	// DeleteLoose causes loose blobs to be deleted after they have been
	// packed into the archive and the archive has been validated.
	DeleteLoose bool

	// DeletionPrecondition is checked before any loose blobs are deleted.
	// When nil, deletion proceeds without additional checks.
	DeletionPrecondition DeletionPrecondition

	// BlobFilter restricts packing to only the specified blob IDs. When nil,
	// all loose blobs not yet in the archive are packed.
	BlobFilter map[string]domain_interfaces.MarklId

	// MaxPackSize overrides the configured max pack size when non-zero.
	MaxPackSize uint64

	// TapWriter emits phase-level TAP test points during packing. When nil,
	// packing is silent (backward compatible for unit tests).
	TapWriter *tap.Writer
}

// PackableArchive is implemented by blob stores that support packing loose
// blobs into archive files.
type PackableArchive interface {
	Pack(options PackOptions) error
}

func tapOk(tw *tap.Writer, desc string) {
	if tw != nil {
		tw.Ok(desc)
	}
}

func tapNotOk(tw *tap.Writer, desc string, err error) {
	if tw != nil {
		tw.NotOk(desc, tap_diagnostics.FromError(err))
	}
}
