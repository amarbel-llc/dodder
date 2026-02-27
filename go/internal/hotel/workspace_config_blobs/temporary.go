package workspace_config_blobs

import "code.linenisgreat.com/dodder/go/internal/golf/repo_configs"

type Temporary struct {
	Defaults repo_configs.DefaultsV1OmitEmpty `toml:"defaults,omitempty"`
}

func (Temporary) temporaryWorkspace() {}

func (blob Temporary) GetDefaults() repo_configs.Defaults {
	return blob.Defaults
}
