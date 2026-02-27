package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type TomlSFTPV0 struct {
	// TODO replace the below with a url scheme
	Host           string `toml:"host"`
	Port           int    `toml:"port,omitempty"`
	User           string `toml:"user"`
	Password       string `toml:"password,omitempty"`
	PrivateKeyPath string `toml:"private-key-path,omitempty"`
	RemotePath     string `toml:"remote-path"`
}

var (
	_ ConfigSFTPRemotePath = &TomlSFTPV0{}
	_ ConfigMutable        = &TomlSFTPV0{}
)

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

func (blobStoreConfig *TomlSFTPV0) SupportsMultiHash() bool {
	return false
}

func (blobStoreConfig *TomlSFTPV0) GetDefaultHashTypeId() string {
	return DefaultHashTypeId
}
