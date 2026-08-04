package genesis_config_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

// TomlV3Common is TomlV2Common plus InstanceId: the repo's uuidv7
// instance identity (RFC-0007 / madder FDR-0010), minted once at
// genesis inside genesis_configs.DefaultWithVersion and immutable
// thereafter — never lazy-minted on read. A pubkey is not
// collision-proof as a sole identity (a hardware-backed key can
// legitimately back more than one repo); the uuid is the identity, the
// pubkey its attestor. Empty for a config decoded from a V2 (legacy)
// repo; legacy repos gain a uuid only via copy-migration, never in
// place.
//
// must be public for toml coding to function
type TomlV3Common struct {
	StoreVersion      store_version.Version `toml:"store-version"`
	RepoId            ids.RepoId            `toml:"id"`
	InventoryListType string                `toml:"inventory_list-type"`
	ObjectSigType     string                `toml:"object-sig-type"`
	InstanceId        markl.Id              `toml:"instance-id,omitempty"`
}

//go:generate tommy generate
type TomlV3Private struct {
	PrivateKey markl.Id `toml:"private-key"`
	TomlV3Common
}

//go:generate tommy generate
type TomlV3Public struct {
	PublicKey markl.Id `toml:"public-key"`
	TomlV3Common
}

var (
	_ ConfigPublic     = &TomlV3Public{}
	_ ConfigPrivate    = &TomlV3Private{}
	_ ConfigInstanceId = &TomlV3Public{}
	_ ConfigInstanceId = &TomlV3Private{}
)

func (config *TomlV3Common) GetInventoryListTypeId() string {
	if config.InventoryListType == "" {
		return ids.TypeInventoryListV1
	} else {
		return config.InventoryListType
	}
}

func (config *TomlV3Common) GetObjectSigMarklTypeId() string {
	if config.ObjectSigType == "" {
		return markl.PurposeObjectSigV2
	} else {
		return config.ObjectSigType
	}
}

func (config *TomlV3Common) GetInstanceId() markl.Id {
	return config.InstanceId
}

func (config *TomlV3Private) GetGenesisConfig() ConfigPrivate {
	return config
}

func (config *TomlV3Private) GetGenesisConfigPublic() ConfigPublic {
	errors.PanicIfError(connectSSHSignerIfNecessary(&config.PrivateKey))
	errors.PanicIfError(connectEcdsaP256SignerIfNecessary(&config.PrivateKey))
	public, err := config.PrivateKey.GetPublicKey(markl.PurposeRepoPrivateKeyV1)
	errors.PanicIfError(err)

	return &TomlV3Public{
		TomlV3Common: config.TomlV3Common,
		PublicKey:    public,
	}
}

func (config *TomlV3Private) GetPrivateKey() mad_domain_interfaces.MarklId {
	errors.PanicIfError(connectSSHSignerIfNecessary(&config.PrivateKey))
	errors.PanicIfError(connectEcdsaP256SignerIfNecessary(&config.PrivateKey))
	return config.PrivateKey
}

func (config *TomlV3Private) GetPrivateKeyMutable() mad_domain_interfaces.MarklIdMutable {
	return &config.PrivateKey
}

func (config *TomlV3Private) GetPublicKey() mad_domain_interfaces.MarklId {
	errors.PanicIfError(connectSSHSignerIfNecessary(&config.PrivateKey))
	errors.PanicIfError(connectEcdsaP256SignerIfNecessary(&config.PrivateKey))
	public, err := config.PrivateKey.GetPublicKey(markl.PurposeRepoPrivateKeyV1)
	errors.PanicIfError(err)
	return public
}

func (config *TomlV3Public) GetGenesisConfig() ConfigPublic {
	return config
}

func (config TomlV3Public) GetPublicKey() mad_domain_interfaces.MarklId {
	return config.PublicKey
}

func (config *TomlV3Common) GetStoreVersion() store_version.Version {
	return config.StoreVersion
}

func (config TomlV3Common) GetRepoId() ids.RepoId {
	return config.RepoId
}

//   __  __       _        _   _
//  |  \/  |_   _| |_ __ _| |_(_) ___  _ __
//  | |\/| | | | | __/ _` | __| |/ _ \| '_ \
//  | |  | | |_| | || (_| | |_| | (_) | | | |
//  |_|  |_|\__,_|\__\__,_|\__|_|\___/|_| |_|
//

func (config *TomlV3Private) SetInventoryListTypeId(value string) {
	config.InventoryListType = value
}

func (config *TomlV3Private) SetObjectSigMarklTypeId(value string) {
	config.ObjectSigType = value
}

func (config *TomlV3Private) SetRepoId(id ids.RepoId) {
	config.RepoId = id
}

func (config *TomlV3Private) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
}
