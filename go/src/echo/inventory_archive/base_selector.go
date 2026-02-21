package inventory_archive

import (
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
)

// BlobMetadata describes a blob candidate for delta packing.
type BlobMetadata struct {
	Id   domain_interfaces.MarklId
	Size uint64
}

// BlobSet provides indexed access to blob metadata without requiring all
// blobs in memory simultaneously.
type BlobSet interface {
	Len() int
	At(index int) BlobMetadata
}

// DeltaAssignments receives base selection results. The packer passes this
// to the strategy, which calls Assign for each blob that should be
// delta-encoded.
type DeltaAssignments interface {
	// Assign records that the blob at blobIndex should be delta-encoded
	// against the blob at baseIndex. Both indices refer to the BlobSet.
	// Not calling Assign for a given index means store it as a full entry.
	Assign(blobIndex, baseIndex int)

	// AssignError reports that the strategy encountered an error for the
	// blob at blobIndex. The packer decides how to handle these.
	AssignError(blobIndex int, err error)
}

// BaseSelector chooses which blobs become deltas and which become bases.
// It reads from blobs and writes results to assignments.
type BaseSelector interface {
	SelectBases(blobs BlobSet, assignments DeltaAssignments)
}
