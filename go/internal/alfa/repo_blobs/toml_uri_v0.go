package repo_blobs

import (
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/values"
)

//go:generate tommy generate
type TomlUriV0 struct {
	PublicKey markl.Id   `toml:"public-key"`
	Uri       values.Uri `toml:"uri"`
}

func (config TomlUriV0) GetPublicKey() mad_domain_interfaces.MarklId {
	return config.PublicKey
}

func (config *TomlUriV0) SetPublicKey(id mad_domain_interfaces.MarklId) {
	config.PublicKey.ResetWithMarklId(id)
}

func (a *TomlUriV0) Reset() {
	a.Uri = values.Uri{}
}

func (a *TomlUriV0) ResetWith(b TomlUriV0) {
	a.Uri = b.Uri
}

func (a TomlUriV0) Equals(b TomlUriV0) bool {
	if a.Uri != b.Uri {
		return false
	}

	return true
}

func (config TomlUriV0) IsRemote() bool {
	return true
}

func (config TomlUriV0) GetUri() values.Uri {
	return config.Uri
}
