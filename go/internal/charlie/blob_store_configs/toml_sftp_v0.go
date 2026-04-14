package blob_store_configs

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

//go:generate tommy generate
type TomlSFTPV0 struct {
	// TODO replace the below with a url scheme
	Host           string `toml:"host"`
	Port           int    `toml:"port,omitempty"`
	User           string `toml:"user"`
	Password       string `toml:"password,omitempty"`
	PrivateKeyPath string `toml:"private-key-path,omitempty"`
	RemotePath     string `toml:"remote-path"`
	KnownHostsFile string `toml:"known-hosts-file,omitempty"`
}

func (*TomlSFTPV0) GetBlobStoreType() string {
	return "sftp"
}

func (blobStoreConfig *TomlSFTPV0) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.StringVar(
		&blobStoreConfig.Host,
		"host",
		blobStoreConfig.Host,
		"SFTP server hostname",
	)

	flagSet.IntVar(
		&blobStoreConfig.Port,
		"port",
		22,
		"SFTP server port",
	)

	flagSet.StringVar(
		&blobStoreConfig.User,
		"user",
		blobStoreConfig.User,
		"SFTP username",
	)

	flagSet.StringVar(
		&blobStoreConfig.Password,
		"password",
		blobStoreConfig.Password,
		"SFTP password",
	)

	flagSet.StringVar(
		&blobStoreConfig.PrivateKeyPath,
		"private-key-path",
		blobStoreConfig.PrivateKeyPath,
		"Path to SSH private key",
	)

	flagSet.StringVar(
		&blobStoreConfig.RemotePath,
		"remote-path",
		blobStoreConfig.RemotePath,
		"Remote path for blob storage",
	)

	flagSet.StringVar(
		&blobStoreConfig.KnownHostsFile,
		"known-hosts-file",
		blobStoreConfig.KnownHostsFile,
		"Path to SSH known_hosts file (default: ~/.ssh/known_hosts)",
	)
}

func (blobStoreConfig *TomlSFTPV0) GetHost() string {
	return blobStoreConfig.Host
}

func (blobStoreConfig *TomlSFTPV0) GetPort() int {
	if blobStoreConfig.Port == 0 {
		return 22
	}
	return blobStoreConfig.Port
}

func (blobStoreConfig *TomlSFTPV0) GetUser() string {
	return blobStoreConfig.User
}

func (blobStoreConfig *TomlSFTPV0) GetPassword() string {
	return blobStoreConfig.Password
}

func (blobStoreConfig *TomlSFTPV0) GetPrivateKeyPath() string {
	return blobStoreConfig.PrivateKeyPath
}

func (blobStoreConfig *TomlSFTPV0) GetRemotePath() string {
	return blobStoreConfig.RemotePath
}

func (blobStoreConfig *TomlSFTPV0) GetKnownHostsFile() string {
	return blobStoreConfig.KnownHostsFile
}

func (blobStoreConfig *TomlSFTPV0) SupportsMultiHash() bool {
	return false
}

func (blobStoreConfig *TomlSFTPV0) GetDefaultHashTypeId() string {
	return DefaultHashTypeId
}
