package genesis_config_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
)

type (
	Config interface {
		GetStoreVersion() store_version.Version
		GetPublicKey() mad_domain_interfaces.MarklId
		GetRepoId() ids.RepoId
		GetInventoryListTypeId() string
		GetObjectSigMarklTypeId() string
	}

	ConfigPublic interface {
		Config
		GetGenesisConfig() ConfigPublic
	}

	ConfigPrivate interface {
		Config
		GetGenesisConfigPublic() ConfigPublic
		GetGenesisConfig() ConfigPrivate
		GetPrivateKey() mad_domain_interfaces.MarklId
	}
)
