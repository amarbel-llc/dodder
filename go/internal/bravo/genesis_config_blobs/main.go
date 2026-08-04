package genesis_config_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
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

	// ConfigInstanceId is implemented by config versions that carry a
	// uuidv7 repo instance identity (RFC-0007 / madder FDR-0010's
	// pattern). Optional: V2 (legacy) configs do not implement it;
	// consumers type-assert. An empty markl.Id means none.
	ConfigInstanceId interface {
		GetInstanceId() markl.Id
	}
)
