package genesis_configs

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

// must be public for toml coding to function
type (
	TomlV2Common struct {
		StoreVersion      store_version.Version `toml:"store-version"`
		_                 string                `toml:"repo-type"`
		RepoId            ids.RepoId            `toml:"id"`
		InventoryListType string                `toml:"inventory_list-type"`
		ObjectSigType     string                `toml:"object-sig-type"`
	}

	TomlV2Private struct {
		PrivateKey markl.Id `toml:"private-key"`
		TomlV2Common
	}

	TomlV2Public struct {
		PublicKey markl.Id `toml:"public-key"`
		TomlV2Common
	}
)

var (
	_ ConfigPublic         = &TomlV2Public{}
	_ ConfigPrivate        = &TomlV2Private{}
	_ ConfigPrivateMutable = &TomlV2Private{}
)

func (config *TomlV2Common) GetInventoryListTypeId() string {
	if config.InventoryListType == "" {
		return ids.TypeInventoryListV1
	} else {
		return config.InventoryListType
	}
}

func (config *TomlV2Common) GetObjectSigMarklTypeId() string {
	if config.ObjectSigType == "" {
		return markl.PurposeObjectSigV2
	} else {
		return config.ObjectSigType
	}
}

func (config *TomlV2Private) GetGenesisConfig() ConfigPrivate {
	return config
}

func (config *TomlV2Private) GetGenesisConfigPublic() ConfigPublic {
	errors.PanicIfError(connectSSHSignerIfNecessary(&config.PrivateKey))
	errors.PanicIfError(connectEcdsaP256SignerIfNecessary(&config.PrivateKey))
	public, err := config.PrivateKey.GetPublicKey(markl.PurposeRepoPrivateKeyV1)
	errors.PanicIfError(err)

	return &TomlV2Public{
		TomlV2Common: config.TomlV2Common,
		PublicKey:    public,
	}
}

func (config *TomlV2Private) GetPrivateKey() domain_interfaces.MarklId {
	errors.PanicIfError(connectSSHSignerIfNecessary(&config.PrivateKey))
	errors.PanicIfError(connectEcdsaP256SignerIfNecessary(&config.PrivateKey))
	return config.PrivateKey
}

func (config *TomlV2Private) GetPrivateKeyMutable() domain_interfaces.MarklIdMutable {
	return &config.PrivateKey
}

func (config *TomlV2Private) GetPublicKey() domain_interfaces.MarklId {
	errors.PanicIfError(connectSSHSignerIfNecessary(&config.PrivateKey))
	errors.PanicIfError(connectEcdsaP256SignerIfNecessary(&config.PrivateKey))
	public, err := config.PrivateKey.GetPublicKey(markl.PurposeRepoPrivateKeyV1)
	errors.PanicIfError(err)
	return public
}

func (config *TomlV2Public) GetGenesisConfig() ConfigPublic {
	return config
}

func (config TomlV2Public) GetPublicKey() domain_interfaces.MarklId {
	return config.PublicKey
}

func (config *TomlV2Common) GetStoreVersion() store_version.Version {
	return config.StoreVersion
}

func (config TomlV2Common) GetRepoId() ids.RepoId {
	return config.RepoId
}

//   __  __       _        _   _
//  |  \/  |_   _| |_ __ _| |_(_) ___  _ __
//  | |\/| | | | | __/ _` | __| |/ _ \| '_ \
//  | |  | | |_| | || (_| | |_| | (_) | | | |
//  |_|  |_|\__,_|\__\__,_|\__|_|\___/|_| |_|
//

func (config *TomlV2Private) SetInventoryListTypeId(value string) {
	config.InventoryListType = value
}

func (config *TomlV2Private) SetObjectSigMarklTypeId(value string) {
	config.ObjectSigType = value
}

func (config *TomlV2Private) SetRepoId(id ids.RepoId) {
	config.RepoId = id
}

func (config *TomlV2Private) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
}
