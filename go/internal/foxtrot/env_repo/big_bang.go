package env_repo

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/genesis_configs"
	"code.linenisgreat.com/madder/go/pkgs/blob_store_configs"
	"code.linenisgreat.com/madder/go/pkgs/blob_store_id"
	"code.linenisgreat.com/madder/go/pkgs/scoped_id"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
)

// Config used to initialize a repo for the first time
type BigBang struct {
	GenesisConfig        *genesis_configs.TypedConfigPrivateMutable
	TypedBlobStoreConfig *blob_store_configs.TypedMutableConfig

	// BlobStoreConfigInit, when non-nil together with a non-empty
	// BlobStoreId, causes Genesis to write this config to disk at the
	// BlobStoreId's blob_store-config path before blob-store discovery
	// is re-run. Used by init-workspace -experimental-repo to install a
	// TomlPointerV1 that resolves to the parent repo's blob store
	// (#200). When set, the pre-flight validation that the BlobStoreId
	// already resolves is skipped — the store is being created right
	// now.
	BlobStoreConfigInit *blob_store_configs.TypedMutableConfig

	InventoryListType ids.TypeStruct

	PrivateKey markl.Id

	Yin  string
	Yang string
	// YinDefault / YangDefault opt into the embedded default zettel-id
	// word lists (zettel_id_provider.Default{Yin,Yang}Reader) for a side
	// whose explicit Yin/Yang path is empty. Set by `dodder init`'s
	// -yin-default / -yang-default flags and by `dodder init-default`.
	YinDefault                    bool
	YangDefault                   bool
	ExcludeDefaultType            bool
	ExcludeDefaultConfig          bool
	ExcludeDefaultPandocTools     bool
	IncludeBuiltinActionableTypes bool
	BlobStoreId                   blob_store_id.Id

	// RepoId is the dodder repo location/name being genesis'd (FDR-0019's
	// scoped_id, the same value command_components_dodder.Genesis's
	// -repo_id-equivalent positional resolves to before OnTheFirstDay
	// calls env_repo.Genesis). FDR-0016 D1: the repo's default blob store
	// is a madder multi named from this id ("default-<name>") so that
	// multiple named repos sharing one XDG scope don't collide on a
	// single flat "default" multi -- only the underlying local write
	// store is shared/reused across repos (see
	// writeBlobStoreConfigIfNecessary).
	RepoId scoped_id.Id
}

func (bigBang *BigBang) SetDefaults() {
	bigBang.GenesisConfig = genesis_configs.Default()
	bigBang.InventoryListType = ids.GetOrPanic(
		ids.TypeInventoryListVCurrent,
	).TypeStruct

	bigBang.TypedBlobStoreConfig = blob_store_configs.Default()
}
