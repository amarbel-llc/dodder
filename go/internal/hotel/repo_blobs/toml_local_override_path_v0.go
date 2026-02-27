package repo_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
)

type TomlLocalOverridePathV0 struct {
	PublicKey    markl.Id `toml:"public-key"`
	OverridePath string   `toml:"override-path"`
}

var (
	_ BlobOverridePath = TomlLocalOverridePathV0{}
	_ BlobMutable      = &TomlLocalOverridePathV0{}
)

func (config TomlLocalOverridePathV0) GetRepoBlob() Blob {
	return config
}

func (config TomlLocalOverridePathV0) GetPublicKey() domain_interfaces.MarklId {
	return config.PublicKey
}

func (config *TomlLocalOverridePathV0) SetPublicKey(id domain_interfaces.MarklId) {
	config.PublicKey.ResetWithMarklId(id)
}

func (config *TomlLocalOverridePathV0) Reset() {
	config.OverridePath = ""
}

func (config *TomlLocalOverridePathV0) ResetWith(b TomlLocalOverridePathV0) {
	config.OverridePath = b.OverridePath
}

func (config TomlLocalOverridePathV0) Equals(b TomlLocalOverridePathV0) bool {
	if config.OverridePath != b.OverridePath {
		return false
	}

	return true
}

func (config TomlLocalOverridePathV0) IsRemote() bool {
	return false
}

func (config TomlLocalOverridePathV0) GetOverridePath() string {
	return config.OverridePath
}
