package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/values"
)

// TODO move to a config_common package
type TomlUriV0 struct {
	Uri values.Uri `toml:"uri"`
}

var _ interfaces.CommandComponentWriter = (*TomlUriV0)(nil)

func (config *TomlUriV0) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	flagSet.Var(
		&config.Uri,
		"uri",
		"SFTP server hostname",
	)
}

func (config *TomlUriV0) GetUri() values.Uri {
	return config.Uri
}
