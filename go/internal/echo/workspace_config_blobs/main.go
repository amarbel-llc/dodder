package workspace_config_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/charlie/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
)

type (
	TypedConfig = triple_hyphen_io.TypedBlob[Config]

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
		domain_interfaces.ConfigDryRunGetter
	}
)

var (
	_ ConfigWithDefaultQueryString = V0{}
	_ ConfigTemporary              = Temporary{}
)
