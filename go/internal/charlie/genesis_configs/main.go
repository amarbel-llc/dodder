package genesis_configs

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	genesis_config_blobs "code.linenisgreat.com/dodder/go/internal/bravo/genesis_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/hyphence"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

type (
	Config        = genesis_config_blobs.Config
	ConfigPublic  = genesis_config_blobs.ConfigPublic
	ConfigPrivate = genesis_config_blobs.ConfigPrivate

	ConfigPrivateMutable interface {
		interfaces.CommandComponentWriter
		ConfigPrivate

		SetInventoryListTypeId(string)
		SetObjectSigMarklTypeId(string)
		SetRepoId(ids.RepoId)
		GetPrivateKeyMutable() mad_domain_interfaces.MarklIdMutable
	}

	TypedConfigPublic         = hyphence.TypedBlob[ConfigPublic]
	TypedConfigPrivate        = hyphence.TypedBlob[ConfigPrivate]
	TypedConfigPrivateMutable = hyphence.TypedBlob[ConfigPrivateMutable]
)

func Default() *TypedConfigPrivateMutable {
	return DefaultWithVersion(
		store_version.VCurrent,
		ids.TypeInventoryListVCurrent,
	)
}

func DefaultWithVersion(
	storeVersion store_version.Version,
	inventoryListTypeString string,
) *TypedConfigPrivateMutable {
	return &TypedConfigPrivateMutable{
		Type: ids.GetOrPanic(
			ids.TypeTomlConfigImmutableV2,
		).TypeStruct.ToMadder(),
		Blob: &TomlV2Private{
			TomlV2Common: TomlV2Common{
				StoreVersion:      storeVersion,
				InventoryListType: inventoryListTypeString,
				ObjectSigType:     markl.PurposeObjectSigV2,
			},
		},
	}
}
