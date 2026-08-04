package genesis_configs

import (
	"code.linenisgreat.com/dodder/go/internal/0/hyphence"
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	genesis_config_blobs "code.linenisgreat.com/dodder/go/internal/bravo/genesis_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/madder/go/pkgs/markl_registrations"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

type (
	Config           = genesis_config_blobs.Config
	ConfigPublic     = genesis_config_blobs.ConfigPublic
	ConfigPrivate    = genesis_config_blobs.ConfigPrivate
	ConfigInstanceId = genesis_config_blobs.ConfigInstanceId

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
	// The repo's uuidv7 instance identity (RFC-0007) is minted here, in
	// the one funnel every fresh genesis config passes through — the
	// dodder analogue of madder's EncodeWithDigest minting (FDR-0010).
	// Configs decoded from disk never pass through this constructor, so
	// a legacy V2 config is never lazy-minted; legacy repos gain a uuid
	// only via copy-migration.
	instanceId, err := markl_registrations.MintInstanceId()
	errors.PanicIfError(err)

	return &TypedConfigPrivateMutable{
		Type: ids.GetOrPanic(
			ids.TypeTomlConfigImmutableV3,
		).TypeStruct.ToMadder(),
		Blob: &TomlV3Private{
			TomlV3Common: TomlV3Common{
				StoreVersion:      storeVersion,
				InventoryListType: inventoryListTypeString,
				ObjectSigType:     markl.PurposeObjectSigV3,
				InstanceId:        instanceId,
			},
		},
	}
}
