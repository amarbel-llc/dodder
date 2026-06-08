package workspace_config_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/hyphence"
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
