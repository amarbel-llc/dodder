package sku

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
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
	UpdateTai          bool
	Validate           bool
}

type LockfileOptions struct {
	AllowTypeFailure bool
	AllowTagFailure  bool
}

func GetStoreOptionsRealizeWithProto() StoreOptions {
	return StoreOptions{
		LockfileOptions: LockfileOptions{
			AllowTypeFailure: true,
			AllowTagFailure:  true,
		},
		ApplyProto: true,
		RunHooks:   true,
		UpdateTai:  true,
	}
}

func GetStoreOptionsRealizeSansProto() StoreOptions {
	return StoreOptions{
		LockfileOptions: LockfileOptions{
			AllowTypeFailure: true,
			AllowTagFailure:  true,
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
