package blob_stores

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
)

// DeletionPrecondition checks whether blobs are safe to delete from the
// loose store. The default implementation always returns nil (safe).
// Future implementations can verify off-host replication before allowing
// deletion.
type DeletionPrecondition interface {
	CheckBlobsSafeToDelete(
		blobs interfaces.SeqError[domain_interfaces.MarklId],
	) error
}

type nopDeletionPrecondition struct{}

func (nopDeletionPrecondition) CheckBlobsSafeToDelete(
	blobs interfaces.SeqError[domain_interfaces.MarklId],
) error {
	return nil
}

func NopDeletionPrecondition() DeletionPrecondition {
	return nopDeletionPrecondition{}
}
