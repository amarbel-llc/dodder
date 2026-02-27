package workspace_config_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/hotel/repo_configs"
)

type V0 struct {
	Defaults repo_configs.DefaultsV1OmitEmpty `toml:"defaults,omitempty"`
	// FileExtensions file_extensions.V1    `toml:"file-extensions"`
	// PrintOptions   options_print.V0      `toml:"cli-output"`
	// Tools          options_tools.Options `toml:"tools"`

	Query string `toml:"query,omitempty"`

	DryRun bool `toml:"dry-run"`
}

func (blob V0) GetDefaults() repo_configs.Defaults {
	return blob.Defaults
}

func (blob V0) GetDefaultQueryString() string {
	return blob.Query
}

func (blob V0) IsDryRun() bool {
	return blob.DryRun
}
