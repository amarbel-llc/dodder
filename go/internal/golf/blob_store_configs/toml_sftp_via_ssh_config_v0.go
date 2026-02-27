package blob_store_configs

import "code.linenisgreat.com/dodder/go/internal/_/interfaces"

type TomlSFTPViaSSHConfigV0 struct {
	TomlUriV0
}

var (
	_ ConfigSFTPRemotePath = TomlSFTPViaSSHConfigV0{}
	_ ConfigMutable        = &TomlSFTPViaSSHConfigV0{}
)

func (TomlSFTPViaSSHConfigV0) GetBlobStoreType() string {
	return "sftp"
}

func (config *TomlSFTPViaSSHConfigV0) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	config.TomlUriV0.SetFlagDefinitions(flagSet)
}

func (config TomlSFTPViaSSHConfigV0) GetRemotePath() string {
	uri := config.TomlUriV0.GetUri()
	return uri.GetUrl().Path
}
