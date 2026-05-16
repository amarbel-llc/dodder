package env_repo

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/genesis_configs"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_configs"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
)

// Config used to initialize a repo for the first time
type BigBang struct {
	GenesisConfig        *genesis_configs.TypedConfigPrivateMutable
	TypedBlobStoreConfig *blob_store_configs.TypedMutableConfig

	InventoryListType ids.TypeStruct

	PrivateKey markl.Id

	Yin                           string
	Yang                          string
	ExcludeDefaultType            bool
	ExcludeDefaultConfig          bool
	IncludeDefaultPandocTools     bool
	IncludeBuiltinActionableTypes bool
	BlobStoreId                   blob_store_id.Id

	// BlobStoreConfigInit, when non-nil AND BlobStoreId is non-empty,
	// causes Genesis to write this config to disk at the BlobStoreId's
	// blob_store-config path before blob-store discovery. Used by
	// init-workspace to install a TomlPointerV1 pointing at the parent
	// repo's blob store.
	BlobStoreConfigInit *blob_store_configs.TypedMutableConfig
}

func (bigBang *BigBang) SetDefaults() {
	bigBang.GenesisConfig = genesis_configs.Default()
	bigBang.InventoryListType = ids.GetOrPanic(
		ids.TypeInventoryListVCurrent,
	).TypeStruct

	bigBang.TypedBlobStoreConfig = blob_store_configs.Default()
}
