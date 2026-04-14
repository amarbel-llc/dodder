package blob_store_configs

import "github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"

//go:generate tommy generate
type TomlSFTPViaSSHConfigV0 struct {
	TomlUriV0
	KnownHostsFile string `toml:"known-hosts-file,omitempty"`
}

func (TomlSFTPViaSSHConfigV0) GetBlobStoreType() string {
	return "sftp"
}

func (config *TomlSFTPViaSSHConfigV0) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	config.TomlUriV0.SetFlagDefinitions(flagSet)

	flagSet.StringVar(
		&config.KnownHostsFile,
		"known-hosts-file",
		config.KnownHostsFile,
		"Path to SSH known_hosts file (default: ~/.ssh/known_hosts)",
	)
}

func (config TomlSFTPViaSSHConfigV0) GetKnownHostsFile() string {
	return config.KnownHostsFile
}

func (config TomlSFTPViaSSHConfigV0) GetRemotePath() string {
	uri := config.TomlUriV0.GetUri()
	return uri.GetUrl().Path
}
