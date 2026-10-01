package sku

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
)

// TODO switch to methods for all
type CommitOptions struct {
	StoreOptions
	ids.RepoId
	ids.Clock
	Proto

	DontAddMissingType bool
}

// TODO switch to methods for all
type StreamIndexOptions struct {
	ForceLatest      bool
	AddToStreamIndex bool
}

// TODO switch to methods for all
type StoreOptions struct {
	LockfileOptions    LockfileOptions
	StreamIndexOptions StreamIndexOptions

	AddToInventoryList bool
	ApplyProto         bool // TODO remove
	ApplyProtoType     bool // TODO remove
	MergeCheckedOut    bool
	RunHooks           bool
	// ProjectFields re-runs only the type's fields-reader projection, without
	// hooks. Projected fields aren't persisted in inventory lists, so reindex
	// must re-project them or type-declared dormancy (FDR 0025) is lost.
	ProjectFields bool
	UpdateTai     bool
	Validate      bool
}

type LockfileOptions struct {
	AllowTypeFailure              bool
	AllowTagFailure               bool
	AllowReferencedObjectFailure  bool
	AllowBlobReferenceTypeFailure bool
}

func GetStoreOptionsRealizeWithProto() StoreOptions {
	return StoreOptions{
		LockfileOptions: LockfileOptions{
			AllowTypeFailure:              true,
			AllowTagFailure:               true,
			AllowReferencedObjectFailure:  true,
			AllowBlobReferenceTypeFailure: true,
		},
		ApplyProto: true,
		RunHooks:   true,
		UpdateTai:  true,
	}
}

func GetStoreOptionsRealizeSansProto() StoreOptions {
	return StoreOptions{
		LockfileOptions: LockfileOptions{
			AllowTypeFailure:              true,
			AllowTagFailure:               true,
			AllowReferencedObjectFailure:  true,
			AllowBlobReferenceTypeFailure: true,
		},
		RunHooks:  true,
		UpdateTai: true,
	}
}

func GetStoreOptionsReindex() StoreOptions {
	return StoreOptions{
		StreamIndexOptions: StreamIndexOptions{
			ForceLatest:      true,
			AddToStreamIndex: true,
		},
		ProjectFields: true,
	}
}

func GetStoreOptionsImport() StoreOptions {
	return StoreOptions{
		AddToInventoryList: true,
		RunHooks:           true,
		Validate:           true,
	}
}

func GetStoreOptionsRemoteTransfer() StoreOptions {
	return StoreOptions{
		AddToInventoryList: true,
	}
}

func GetStoreOptionsUpdate() StoreOptions {
	return StoreOptions{
		AddToInventoryList: true,
		RunHooks:           true,
		UpdateTai:          true,
		Validate:           true,
	}
}

func GetStoreOptionsCreate() StoreOptions {
	return StoreOptions{
		AddToInventoryList: true,
		RunHooks:           true,
		ApplyProto:         true,
		UpdateTai:          true,
		Validate:           true,
	}
}
