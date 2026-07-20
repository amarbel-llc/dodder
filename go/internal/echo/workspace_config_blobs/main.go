package workspace_config_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/0/hyphence"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
)

type (
	TypedConfig = hyphence.TypedBlob[Config]

	Config interface {
		GetDefaults() repo_configs.Defaults
	}

	ConfigWithRepo interface {
		GetRepoConfig() repo_configs.ConfigOverlay
	}

	ConfigTemporary interface {
		Config
		temporaryWorkspace()
	}

	ConfigWithDefaultQueryString interface {
		Config
		GetDefaultQueryString() string
	}

	ConfigWithDryRun interface {
		Config
		mad_domain_interfaces.ConfigDryRunGetter
	}

	ConfigWithParentPath interface {
		Config
		GetParentPath() string
	}

	ConfigWithParentPubkey interface {
		Config
		GetParentPubkey() string
	}

	ConfigWithSyncBaseline interface {
		Config
		GetSyncTai() string
		GetSyncDigest() string
	}

	ConfigWithHaustoria interface {
		Config
		GetHaustoriaConfig() HaustoriaConfig
	}

	ConfigWithIgnore interface {
		Config
		GetIgnorePatterns() []string
	}
)

var _ ConfigTemporary = Temporary{}
