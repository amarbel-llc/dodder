package blob_stores

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/hotel/tap_diagnostics"
	tap "github.com/amarbel-llc/tap-dancer/go"
)

// PackOptions controls the behavior of the Pack operation.
type PackOptions struct {
	// Context supports cancellation of the pack operation via signals or
	// memory-exhaustion monitoring. When nil, packing runs without
	// cancellation support.
	Context interfaces.ActiveContext

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

	// SkipMissingBlobs causes unreadable loose blobs to be skipped with a
	// TAP comment instead of aborting the pack. When false, an unreadable
	// blob emits a not-ok test point and stops packing.
	SkipMissingBlobs bool

	// TapWriter emits phase-level TAP test points during packing. When nil,
	// packing is silent (backward compatible for unit tests).
	TapWriter *tap.Writer
}

// PackableArchive is implemented by blob stores that support packing loose
// blobs into archive files.
type PackableArchive interface {
	Pack(options PackOptions) error
}

func packContextCancelled(ctx interfaces.ActiveContext) error {
	if ctx == nil {
		return nil
	}

	select {
	default:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
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

func tapComment(tw *tap.Writer, msg string) {
	if tw != nil {
		tw.Comment(msg)
	}
}
