package repo_blobs

import (
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
)

//go:generate tommy generate
type TomlLocalOverridePathV0 struct {
	PublicKey    markl.Id `toml:"public-key"`
	OverridePath string   `toml:"override-path"`
}

func (config TomlLocalOverridePathV0) GetPublicKey() mad_domain_interfaces.MarklId {
	return config.PublicKey
}

func (config *TomlLocalOverridePathV0) SetPublicKey(id mad_domain_interfaces.MarklId) {
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
